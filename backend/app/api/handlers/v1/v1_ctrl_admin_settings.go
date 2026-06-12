package v1

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hay-kot/httpkit/errchain"
	"github.com/hay-kot/httpkit/server"
	"github.com/rs/zerolog/log"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/settings"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
)

// AdminSettingsOut is the full site settings document. Secret fields are
// redacted to the "[REDACTED]" sentinel; sending the sentinel back on update
// keeps the stored value.
type AdminSettingsOut struct {
	Settings settings.Resolved `json:"settings"`
}

func (ctrl *V1Controller) adminSettingsOut() (AdminSettingsOut, error) {
	if ctrl.settings == nil {
		return AdminSettingsOut{}, validate.NewRequestError(errors.New("site settings not available"), http.StatusNotFound)
	}
	return AdminSettingsOut{
		Settings: ctrl.settings.Get(),
	}, nil
}

// HandleAdminSettingsGet godoc
//
//	@Summary	Get site settings
//	@Tags		Admin Settings
//	@Produce	json
//	@Success	200	{object}	AdminSettingsOut
//	@Router		/v1/admin/settings [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandleAdminSettingsGet() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		out, err := ctrl.adminSettingsOut()
		if err != nil {
			return err
		}
		w.Header().Set("Cache-Control", "no-store")
		return server.JSON(w, http.StatusOK, out)
	}
}

// HandleAdminSettingsUpdate godoc
//
//	@Summary	Update one site settings section
//	@Tags		Admin Settings
//	@Accept		json
//	@Produce	json
//	@Param		section	path	string					true	"Section name"
//	@Param		payload	body	map[string]interface{}	true	"Sparse section override"
//	@Success	200	{object}	AdminSettingsOut
//	@Router		/v1/admin/settings/{section} [PUT]
//	@Security	Bearer
func (ctrl *V1Controller) HandleAdminSettingsUpdate() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if ctrl.isDemo {
			return validate.NewRequestError(errors.New("settings are read-only in demo mode"), http.StatusForbidden)
		}
		if ctrl.settings == nil {
			return validate.NewRequestError(errors.New("site settings not available"), http.StatusNotFound)
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64*1024))
		if err != nil {
			return validate.NewRequestError(err, http.StatusBadRequest)
		}

		section := chi.URLParam(r, "section")
		err = ctrl.settings.UpdateSection(r.Context(), section, body)
		switch {
		case errors.Is(err, settings.ErrUnknownSection):
			return validate.NewRequestError(fmt.Errorf("unknown settings section %q", section), http.StatusNotFound)
		case errors.Is(err, settings.ErrInvalidPayload):
			return validate.NewRequestError(err, http.StatusBadRequest)
		case err != nil:
			log.Err(err).Str("section", section).Msg("failed to update site settings")
			return validate.NewRequestError(err, http.StatusInternalServerError)
		}

		out, err := ctrl.adminSettingsOut()
		if err != nil {
			return err
		}
		w.Header().Set("Cache-Control", "no-store")
		return server.JSON(w, http.StatusOK, out)
	}
}

// HandleAdminSettingsReset godoc
//
//	@Summary	Reset one site settings section to default values
//	@Tags		Admin Settings
//	@Produce	json
//	@Param		section	path	string	true	"Section name"
//	@Success	200	{object}	AdminSettingsOut
//	@Router		/v1/admin/settings/{section} [DELETE]
//	@Security	Bearer
func (ctrl *V1Controller) HandleAdminSettingsReset() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if ctrl.isDemo {
			return validate.NewRequestError(errors.New("settings are read-only in demo mode"), http.StatusForbidden)
		}
		if ctrl.settings == nil {
			return validate.NewRequestError(errors.New("site settings not available"), http.StatusNotFound)
		}

		section := chi.URLParam(r, "section")
		err := ctrl.settings.ResetSection(r.Context(), section)
		switch {
		case errors.Is(err, settings.ErrUnknownSection):
			return validate.NewRequestError(fmt.Errorf("unknown settings section %q", section), http.StatusNotFound)
		case err != nil:
			log.Err(err).Str("section", section).Msg("failed to reset site settings")
			return validate.NewRequestError(err, http.StatusInternalServerError)
		}

		out, err := ctrl.adminSettingsOut()
		if err != nil {
			return err
		}
		w.Header().Set("Cache-Control", "no-store")
		return server.JSON(w, http.StatusOK, out)
	}
}

// HandleAdminSettingsAlgoliaReindex godoc
//
//	@Summary	Trigger a full Algolia reindex
//	@Tags		Admin Settings
//	@Produce	json
//	@Success	202
//	@Router		/v1/admin/settings/algolia/reindex [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandleAdminSettingsAlgoliaReindex() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		// The integration not existing for this caller is indistinguishable
		// from the route not existing.
		if ctrl.algoliaReindex == nil || !ctrl.runtime().Algolia.Enabled {
			return validate.NewRequestError(errors.New("algolia integration is not enabled"), http.StatusNotFound)
		}

		go ctrl.algoliaReindex()
		return server.JSON(w, http.StatusAccepted, map[string]string{"status": "reindex started"})
	}
}
