package v1

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/hay-kot/httpkit/errchain"
	"github.com/hay-kot/httpkit/server"
	"github.com/rs/zerolog/log"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/ai"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
	"github.com/sysadminsmedia/homebox/backend/internal/web/adapters"
)

type (
	AIStatus struct {
		Enabled bool `json:"enabled"`
	}

	AIAnalyzeResult struct {
		Items []ai.AnalyzedItem `json:"items"`
	}

	AISuggestRequest struct {
		Overwrite bool `json:"overwrite"`
	}

	AISuggestResult struct {
		Suggestions []ai.FieldSuggestion `json:"suggestions"`
	}
)

// aiError maps AI service failures onto HTTP statuses: a disabled integration
// reads as 404 (the feature does not exist), bad inputs as 422, provider
// failures as 502.
func aiError(err error) error {
	switch {
	case errors.Is(err, ai.ErrDisabled):
		return validate.NewRequestError(errors.New("ai integration is not enabled"), http.StatusNotFound)
	case errors.Is(err, ai.ErrNoPhotos):
		return validate.NewRequestError(err, http.StatusUnprocessableEntity)
	case ent.IsNotFound(err):
		return validate.NewRequestError(err, http.StatusNotFound)
	default:
		log.Err(err).Msg("ai request failed")
		return validate.NewRequestError(errors.New("ai request failed"), http.StatusBadGateway)
	}
}

// HandleAIStatus godoc
//
//	@Summary		AI integration status
//	@Description	Reports whether AI features are enabled on this instance
//	@Tags			AI
//	@Produce		json
//	@Success		200	{object}	AIStatus
//	@Router			/v1/ai/status [GET]
//	@Security		Bearer
func (ctrl *V1Controller) HandleAIStatus() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		return server.JSON(w, http.StatusOK, AIStatus{Enabled: ctrl.svc.AI.Enabled()})
	}
}

// HandleAIAnalyze godoc
//
//	@Summary		Detect items in photos
//	@Description	Runs vision analysis over uploaded photos and returns detected inventory items
//	@Tags			AI
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			images	formData	file	true	"Photos to analyze (repeatable, max 8)"
//	@Param			options	formData	string	false	"JSON-encoded ai.AnalyzeOptions"
//	@Success		200	{object}	AIAnalyzeResult
//	@Router			/v1/ai/analyze [POST]
//	@Security		Bearer
func (ctrl *V1Controller) HandleAIAnalyze() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		ctx := services.NewContext(r.Context())

		if err := r.ParseMultipartForm(ctrl.maxUploadSize << 20); err != nil {
			return validate.NewRequestError(errors.New("failed to parse multipart form"), http.StatusUnprocessableEntity)
		}

		var opts ai.AnalyzeOptions
		if raw := r.FormValue("options"); raw != "" {
			if err := json.Unmarshal([]byte(raw), &opts); err != nil {
				return validate.NewRequestError(errors.New("invalid options payload"), http.StatusUnprocessableEntity)
			}
		}

		files := r.MultipartForm.File["images"]
		if len(files) == 0 {
			return validate.NewRequestError(errors.New("no images supplied"), http.StatusUnprocessableEntity)
		}
		if len(files) > ai.MaxImages {
			files = files[:ai.MaxImages]
		}

		images := make([][]byte, 0, len(files))
		for _, header := range files {
			file, err := header.Open()
			if err != nil {
				return validate.NewRequestError(errors.New("failed to read image"), http.StatusUnprocessableEntity)
			}
			raw, err := io.ReadAll(file)
			_ = file.Close()
			if err != nil {
				return validate.NewRequestError(errors.New("failed to read image"), http.StatusUnprocessableEntity)
			}
			images = append(images, raw)
		}

		items, err := ctrl.svc.AI.AnalyzeImages(r.Context(), ctx.GID, images, opts)
		if err != nil {
			return aiError(err)
		}
		return server.JSON(w, http.StatusOK, AIAnalyzeResult{Items: items})
	}
}

// HandleAISuggest godoc
//
//	@Summary		Suggest field values from an item's photos
//	@Description	Analyzes an item's photo attachments and proposes values for its catalog fields
//	@Tags			AI
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string				true	"Entity ID"
//	@Param			payload	body	AISuggestRequest	true	"Suggestion options"
//	@Success		200	{object}	AISuggestResult
//	@Router			/v1/ai/entities/{id}/suggest [POST]
//	@Security		Bearer
func (ctrl *V1Controller) HandleAISuggest() errchain.HandlerFunc {
	fn := func(r *http.Request, ID uuid.UUID, body AISuggestRequest) (AISuggestResult, error) {
		ctx := services.NewContext(r.Context())

		suggestions, err := ctrl.svc.AI.SuggestForItem(ctx, ctx.GID, ID, body.Overwrite)
		if err != nil {
			return AISuggestResult{}, aiError(err)
		}
		return AISuggestResult{Suggestions: suggestions}, nil
	}

	return adapters.ActionID("id", fn, http.StatusOK)
}
