package repo

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/schema"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/sitesetting"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/theme"
	"gocloud.dev/blob"
)

// ThemeRepository persists admin-created instance themes and the site-wide
// active theme pointer (stored as the "theming" site settings row). Uploaded
// branding images live in the same blob bucket as attachments, under
// theming/<theme-id>/.
type ThemeRepository struct {
	db *ent.Client
	// attachments supplies the blob connection string and prefix-path
	// handling so theme assets share the configured storage backend.
	attachments *AttachmentRepo
}

func NewThemeRepository(db *ent.Client, attachments *AttachmentRepo) *ThemeRepository {
	return &ThemeRepository{db: db, attachments: attachments}
}

// themingSettingsKey is the site_settings row holding the active pointer.
const themingSettingsKey = "theming"

// DefaultActiveTheme is the pointer value when no row exists.
const DefaultActiveTheme = "builtin:homebox"

// ThemeAssetKinds are the uploadable branding image slots.
var ThemeAssetKinds = []string{"nav-logo", "sidebar-logo", "login-icon"}

// coreColorKeys are the exact keys every theme's colors document must carry.
var coreColorKeys = []string{"background", "foreground", "primary", "secondary", "accent", "destructive"}

var hexColorRx = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

var (
	ErrThemeInvalidColors    = errors.New("colors must contain exactly background, foreground, primary, secondary, accent and destructive as #rrggbb values")
	ErrThemeActive           = errors.New("theme is active and cannot be deleted")
	ErrThemeInvalidPointer   = errors.New("active theme must be builtin:<slug> or custom:<uuid>")
	ErrThemeUnknownAssetKind = errors.New("unknown theme asset kind")
)

type (
	ThemeCreate struct {
		Name     string               `json:"name" validate:"required,max=255"`
		Colors   map[string]string    `json:"colors" validate:"required"`
		Radius   string               `json:"radius"`
		FontSans string               `json:"fontSans"`
		FontMono string               `json:"fontMono"`
		Branding schema.ThemeBranding `json:"branding"`
	}

	ThemeUpdate struct {
		Name     string               `json:"name" validate:"required,max=255"`
		Colors   map[string]string    `json:"colors" validate:"required"`
		Radius   string               `json:"radius"`
		FontSans string               `json:"fontSans"`
		FontMono string               `json:"fontMono"`
		Branding schema.ThemeBranding `json:"branding"`
	}

	// ThemeAssets reports which branding images have been uploaded.
	ThemeAssets struct {
		NavLogo     bool `json:"navLogo"`
		SidebarLogo bool `json:"sidebarLogo"`
		LoginIcon   bool `json:"loginIcon"`
	}

	ThemeOut struct {
		ID        uuid.UUID            `json:"id"`
		CreatedAt time.Time            `json:"createdAt"`
		UpdatedAt time.Time            `json:"updatedAt"`
		Name      string               `json:"name"`
		Colors    map[string]string    `json:"colors"`
		Radius    string               `json:"radius"`
		FontSans  string               `json:"fontSans"`
		FontMono  string               `json:"fontMono"`
		Branding  schema.ThemeBranding `json:"branding"`
		Assets    ThemeAssets          `json:"assets"`
	}

	// ThemingSettings is the JSON document in the "theming" site settings row.
	ThemingSettings struct {
		Active string `json:"active"`
	}
)

func mapThemeOut(t *ent.Theme) ThemeOut {
	return ThemeOut{
		ID:        t.ID,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
		Name:      t.Name,
		Colors:    t.Colors,
		Radius:    t.Radius,
		FontSans:  t.FontSans,
		FontMono:  t.FontMono,
		Branding:  t.Branding,
		Assets: ThemeAssets{
			NavLogo:     t.NavLogoPath != "",
			SidebarLogo: t.SidebarLogoPath != "",
			LoginIcon:   t.LoginIconPath != "",
		},
	}
}

func validateThemeColors(colors map[string]string) error {
	if len(colors) != len(coreColorKeys) {
		return ErrThemeInvalidColors
	}
	for _, key := range coreColorKeys {
		if !hexColorRx.MatchString(colors[key]) {
			return ErrThemeInvalidColors
		}
	}
	return nil
}

func (r *ThemeRepository) GetAll(ctx context.Context) ([]ThemeOut, error) {
	rows, err := r.db.Theme.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]ThemeOut, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapThemeOut(row))
	}
	return out, nil
}

func (r *ThemeRepository) Get(ctx context.Context, id uuid.UUID) (ThemeOut, error) {
	row, err := r.db.Theme.Get(ctx, id)
	if err != nil {
		return ThemeOut{}, err
	}
	return mapThemeOut(row), nil
}

func (r *ThemeRepository) Create(ctx context.Context, data ThemeCreate) (ThemeOut, error) {
	if err := validateThemeColors(data.Colors); err != nil {
		return ThemeOut{}, err
	}

	row, err := r.db.Theme.Create().
		SetName(data.Name).
		SetColors(data.Colors).
		SetRadius(cmp.Or(data.Radius, "0.5rem")).
		SetFontSans(data.FontSans).
		SetFontMono(data.FontMono).
		SetBranding(data.Branding).
		Save(ctx)
	if err != nil {
		return ThemeOut{}, err
	}
	return mapThemeOut(row), nil
}

func (r *ThemeRepository) Update(ctx context.Context, id uuid.UUID, data ThemeUpdate) (ThemeOut, error) {
	if err := validateThemeColors(data.Colors); err != nil {
		return ThemeOut{}, err
	}

	row, err := r.db.Theme.UpdateOneID(id).
		SetName(data.Name).
		SetColors(data.Colors).
		SetRadius(cmp.Or(data.Radius, "0.5rem")).
		SetFontSans(data.FontSans).
		SetFontMono(data.FontMono).
		SetBranding(data.Branding).
		Save(ctx)
	if err != nil {
		return ThemeOut{}, err
	}
	return mapThemeOut(row), nil
}

// Delete removes a theme and its uploaded assets. Deleting the active theme
// is rejected so the instance never points at a missing theme.
func (r *ThemeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	active, err := r.GetActive(ctx)
	if err != nil {
		return err
	}
	if active == "custom:"+id.String() {
		return ErrThemeActive
	}

	row, err := r.db.Theme.Get(ctx, id)
	if err != nil {
		return err
	}

	for _, path := range []string{row.NavLogoPath, row.SidebarLogoPath, row.LoginIconPath} {
		if path != "" {
			r.deleteBlob(ctx, path)
		}
	}

	return r.db.Theme.DeleteOneID(id).Exec(ctx)
}

// GetActive returns the active theme pointer ("builtin:<slug>" or
// "custom:<uuid>"), defaulting when no row exists.
func (r *ThemeRepository) GetActive(ctx context.Context) (string, error) {
	row, err := r.db.SiteSetting.Query().
		Where(sitesetting.Key(themingSettingsKey)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return DefaultActiveTheme, nil
		}
		return "", err
	}

	var settings ThemingSettings
	if err := json.Unmarshal(row.Value, &settings); err != nil || settings.Active == "" {
		return DefaultActiveTheme, nil
	}
	return settings.Active, nil
}

// GetActiveTheme resolves the active pointer to its custom theme row, or nil
// when a built-in theme is active.
func (r *ThemeRepository) GetActiveTheme(ctx context.Context) (string, *ThemeOut, error) {
	active, err := r.GetActive(ctx)
	if err != nil {
		return "", nil, err
	}

	id, ok := customThemeID(active)
	if !ok {
		return active, nil, nil
	}

	row, err := r.db.Theme.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			// The pointed-at theme vanished (e.g. raw DB edit); fall back.
			return DefaultActiveTheme, nil, nil
		}
		return "", nil, err
	}

	out := mapThemeOut(row)
	return active, &out, nil
}

// SetActive validates and stores the active theme pointer.
func (r *ThemeRepository) SetActive(ctx context.Context, active string) error {
	switch {
	case strings.HasPrefix(active, "builtin:"):
		if len(active) == len("builtin:") {
			return ErrThemeInvalidPointer
		}
	case strings.HasPrefix(active, "custom:"):
		id, ok := customThemeID(active)
		if !ok {
			return ErrThemeInvalidPointer
		}
		exists, err := r.db.Theme.Query().Where(theme.ID(id)).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return ErrThemeInvalidPointer
		}
	default:
		return ErrThemeInvalidPointer
	}

	value, err := json.Marshal(ThemingSettings{Active: active})
	if err != nil {
		return err
	}

	n, err := r.db.SiteSetting.Update().
		Where(sitesetting.Key(themingSettingsKey)).
		SetValue(value).
		Save(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	return r.db.SiteSetting.Create().
		SetKey(themingSettingsKey).
		SetValue(value).
		Exec(ctx)
}

func customThemeID(active string) (uuid.UUID, bool) {
	raw, ok := strings.CutPrefix(active, "custom:")
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// AssetPath returns the stored blob path for one asset slot of a theme,
// empty when nothing is uploaded.
func (r *ThemeRepository) AssetPath(ctx context.Context, id uuid.UUID, kind string) (string, error) {
	row, err := r.db.Theme.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return assetPathByKind(row, kind)
}

// ActiveAssetPath resolves the asset path for the active custom theme; empty
// when a built-in theme is active or the slot has no upload.
func (r *ThemeRepository) ActiveAssetPath(ctx context.Context, kind string) (string, error) {
	active, err := r.GetActive(ctx)
	if err != nil {
		return "", err
	}
	id, ok := customThemeID(active)
	if !ok {
		return "", nil
	}

	row, err := r.db.Theme.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}
	return assetPathByKind(row, kind)
}

func assetPathByKind(row *ent.Theme, kind string) (string, error) {
	switch kind {
	case "nav-logo":
		return row.NavLogoPath, nil
	case "sidebar-logo":
		return row.SidebarLogoPath, nil
	case "login-icon":
		return row.LoginIconPath, nil
	default:
		return "", ErrThemeUnknownAssetKind
	}
}

// SetAsset stores an uploaded branding image and records its path on the
// theme, replacing (and removing) any previous upload in that slot.
func (r *ThemeRepository) SetAsset(ctx context.Context, id uuid.UUID, kind string, filename string, content io.Reader) (ThemeOut, error) {
	row, err := r.db.Theme.Get(ctx, id)
	if err != nil {
		return ThemeOut{}, err
	}

	oldPath, err := assetPathByKind(row, kind)
	if err != nil {
		return ThemeOut{}, err
	}

	data, err := io.ReadAll(content)
	if err != nil {
		return ThemeOut{}, err
	}

	relativePath := fmt.Sprintf("theming/%s/%s-%s", id.String(), kind, filename)

	bucket, err := blob.OpenBucket(ctx, r.attachments.GetConnString())
	if err != nil {
		return ThemeOut{}, err
	}
	defer closeBucket(bucket)

	if err := bucket.WriteAll(ctx, r.attachments.GetFullPath(relativePath), data, nil); err != nil {
		return ThemeOut{}, err
	}

	update := r.db.Theme.UpdateOneID(id)
	switch kind {
	case "nav-logo":
		update.SetNavLogoPath(relativePath)
	case "sidebar-logo":
		update.SetSidebarLogoPath(relativePath)
	case "login-icon":
		update.SetLoginIconPath(relativePath)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		return ThemeOut{}, err
	}

	if oldPath != "" && oldPath != relativePath {
		r.deleteBlob(ctx, oldPath)
	}

	return mapThemeOut(updated), nil
}

// DeleteAsset removes an uploaded branding image and clears its slot.
func (r *ThemeRepository) DeleteAsset(ctx context.Context, id uuid.UUID, kind string) (ThemeOut, error) {
	row, err := r.db.Theme.Get(ctx, id)
	if err != nil {
		return ThemeOut{}, err
	}

	path, err := assetPathByKind(row, kind)
	if err != nil {
		return ThemeOut{}, err
	}

	update := r.db.Theme.UpdateOneID(id)
	switch kind {
	case "nav-logo":
		update.SetNavLogoPath("")
	case "sidebar-logo":
		update.SetSidebarLogoPath("")
	case "login-icon":
		update.SetLoginIconPath("")
	}

	updated, err := update.Save(ctx)
	if err != nil {
		return ThemeOut{}, err
	}

	if path != "" {
		r.deleteBlob(ctx, path)
	}

	return mapThemeOut(updated), nil
}

// FullAssetPath exposes the blob key for a stored relative path so handlers
// can stream the file.
func (r *ThemeRepository) FullAssetPath(relativePath string) string {
	return r.attachments.GetFullPath(relativePath)
}

// ConnString exposes the blob connection string for handlers.
func (r *ThemeRepository) ConnString() string {
	return r.attachments.GetConnString()
}

func (r *ThemeRepository) deleteBlob(ctx context.Context, relativePath string) {
	bucket, err := blob.OpenBucket(ctx, r.attachments.GetConnString())
	if err != nil {
		log.Err(err).Msg("failed to open bucket to delete theme asset")
		return
	}
	defer closeBucket(bucket)

	if err := bucket.Delete(ctx, r.attachments.GetFullPath(relativePath)); err != nil {
		log.Err(err).Str("path", relativePath).Msg("failed to delete theme asset blob")
	}
}

func closeBucket(bucket *blob.Bucket) {
	if err := bucket.Close(); err != nil {
		log.Err(err).Msg("failed to close bucket")
	}
}
