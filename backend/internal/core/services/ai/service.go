// Package ai provides vision-LLM features: detecting items in capture photos
// and suggesting field values for existing items from their photos. All calls
// go through the instance-wide OpenAI-compatible endpoint configured in the
// admin settings; the model never sees anything beyond the photos and prompts.
package ai

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"strings"

	_ "image/png"

	_ "github.com/gen2brain/avif"
	_ "github.com/gen2brain/heic"
	_ "github.com/gen2brain/webp"

	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	"github.com/sysadminsmedia/homebox/backend/pkgs/llm"
	"golang.org/x/image/draw"
)

const (
	// MaxImages caps photos per analyze request; more adds cost without
	// improving detection.
	MaxImages = 8
	// maxImageEdge is the longest edge images are downscaled to before being
	// sent to the model; matches common provider vision input limits.
	maxImageEdge = 1568
	jpegQuality  = 85
)

var (
	// ErrDisabled is returned when the AI section is disabled or incomplete.
	ErrDisabled = errors.New("ai: not enabled")
	// ErrNoPhotos is returned when an item has no photo attachments to analyze.
	ErrNoPhotos = errors.New("ai: item has no photos")
)

// Service implements the AI features over the configured LLM endpoint.
type Service struct {
	repos *repo.AllRepos
	// conf reads the effective AI configuration on every call so admin
	// settings changes apply without a restart.
	conf func() config.AIConf
}

// NewService builds the AI service. conf must not be nil.
func NewService(repos *repo.AllRepos, conf func() config.AIConf) *Service {
	return &Service{repos: repos, conf: conf}
}

// Enabled reports whether AI features are configured and turned on.
func (s *Service) Enabled() bool {
	c := s.conf()
	return c.Enabled && c.BaseURL != "" && c.Model != ""
}

// client returns a configured LLM client or ErrDisabled.
func (s *Service) client() (*llm.Client, string, error) {
	c := s.conf()
	if !c.Enabled || c.BaseURL == "" || c.Model == "" {
		return nil, "", ErrDisabled
	}
	return llm.NewClient(c.BaseURL, c.APIKey, c.Model), c.ExtraInstructions, nil
}

// prepareImage decodes raw upload bytes (JPEG/PNG/WebP/HEIC/AVIF), downscales
// to the model input size, and re-encodes as JPEG.
func prepareImage(raw []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("ai: decoding image: %w", err)
	}

	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	longest := max(w, h)
	if longest > maxImageEdge {
		scale := float64(maxImageEdge) / float64(longest)
		dst := image.NewRGBA(image.Rect(0, 0, int(float64(w)*scale), int(float64(h)*scale)))
		draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
		img = dst
	}

	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, fmt.Errorf("ai: encoding image: %w", err)
	}
	return out.Bytes(), nil
}

// emptyish reports whether a model output carries no usable value.
func emptyish(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "unknown", "n/a", "none", "null":
		return true
	}
	return false
}
