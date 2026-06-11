package v1

import (
	"errors"
	"net/http"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hay-kot/httpkit/errchain"
	"github.com/hay-kot/httpkit/server"
	"github.com/rs/zerolog/log"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
	"github.com/sysadminsmedia/homebox/backend/internal/web/adapters"
	"gocloud.dev/blob"
)

// themeAssetExtensions are the upload formats accepted for branding images.
var themeAssetExtensions = []string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".svg", ".ico"}

var themeAssetContentTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".gif":  "image/gif",
	".svg":  "image/svg+xml",
	".ico":  "image/x-icon",
}

// mapThemeError converts theme repository sentinel errors to request errors.
func mapThemeError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repo.ErrThemeActive):
		return validate.NewRequestError(err, http.StatusConflict)
	case errors.Is(err, repo.ErrThemeInvalidColors),
		errors.Is(err, repo.ErrThemeInvalidPointer),
		errors.Is(err, repo.ErrThemeUnknownAssetKind):
		return validate.NewRequestError(err, http.StatusUnprocessableEntity)
	case ent.IsNotFound(err):
		return validate.NewRequestError(err, http.StatusNotFound)
	default:
		return err
	}
}

func themeAssetKind(r *http.Request) (string, error) {
	kind := chi.URLParam(r, "kind")
	if !slices.Contains(repo.ThemeAssetKinds, kind) {
		return "", validate.NewRequestError(repo.ErrThemeUnknownAssetKind, http.StatusNotFound)
	}
	return kind, nil
}

// HandleThemesGetAll godoc
//
//	@Summary	Get All Themes
//	@Tags		Theming
//	@Produce	json
//	@Success	200	{object}	[]repo.ThemeOut
//	@Router		/v1/themes [Get]
//	@Security	Bearer
func (ctrl *V1Controller) HandleThemesGetAll() errchain.HandlerFunc {
	fn := func(r *http.Request) ([]repo.ThemeOut, error) {
		return ctrl.repo.Themes.GetAll(r.Context())
	}

	return adapters.Command(fn, http.StatusOK)
}

// HandleThemeGet godoc
//
//	@Summary	Get Theme
//	@Tags		Theming
//	@Produce	json
//	@Param		id	path		string	true	"Theme ID"
//	@Success	200	{object}	repo.ThemeOut
//	@Router		/v1/themes/{id} [Get]
//	@Security	Bearer
func (ctrl *V1Controller) HandleThemeGet() errchain.HandlerFunc {
	fn := func(r *http.Request, id uuid.UUID) (repo.ThemeOut, error) {
		out, err := ctrl.repo.Themes.Get(r.Context(), id)
		return out, mapThemeError(err)
	}

	return adapters.CommandID("id", fn, http.StatusOK)
}

// HandleThemeCreate godoc
//
//	@Summary	Create Theme
//	@Tags		Theming
//	@Produce	json
//	@Param		payload	body		repo.ThemeCreate	true	"Theme Data"
//	@Success	201		{object}	repo.ThemeOut
//	@Router		/v1/themes [Post]
//	@Security	Bearer
func (ctrl *V1Controller) HandleThemeCreate() errchain.HandlerFunc {
	fn := func(r *http.Request, body repo.ThemeCreate) (repo.ThemeOut, error) {
		out, err := ctrl.repo.Themes.Create(r.Context(), body)
		return out, mapThemeError(err)
	}

	return adapters.Action(fn, http.StatusCreated)
}

// HandleThemeUpdate godoc
//
//	@Summary	Update Theme
//	@Tags		Theming
//	@Produce	json
//	@Param		id		path		string			true	"Theme ID"
//	@Param		payload	body		repo.ThemeUpdate	true	"Theme Data"
//	@Success	200		{object}	repo.ThemeOut
//	@Router		/v1/themes/{id} [Put]
//	@Security	Bearer
func (ctrl *V1Controller) HandleThemeUpdate() errchain.HandlerFunc {
	fn := func(r *http.Request, id uuid.UUID, body repo.ThemeUpdate) (repo.ThemeOut, error) {
		out, err := ctrl.repo.Themes.Update(r.Context(), id, body)
		return out, mapThemeError(err)
	}

	return adapters.ActionID("id", fn, http.StatusOK)
}

// HandleThemeDelete godoc
//
//	@Summary	Delete Theme
//	@Tags		Theming
//	@Produce	json
//	@Param		id	path	string	true	"Theme ID"
//	@Success	204
//	@Failure	409	{object}	validate.ErrorResponse	"theme is active"
//	@Router		/v1/themes/{id} [Delete]
//	@Security	Bearer
func (ctrl *V1Controller) HandleThemeDelete() errchain.HandlerFunc {
	fn := func(r *http.Request, id uuid.UUID) (any, error) {
		err := ctrl.repo.Themes.Delete(r.Context(), id)
		return nil, mapThemeError(err)
	}

	return adapters.CommandID("id", fn, http.StatusNoContent)
}

// HandleThemingActive godoc
//
//	@Summary	Get Active Theme Pointer
//	@Tags		Theming
//	@Produce	json
//	@Success	200	{object}	repo.ThemingSettings
//	@Router		/v1/theming/active [Get]
//	@Security	Bearer
func (ctrl *V1Controller) HandleThemingActive() errchain.HandlerFunc {
	fn := func(r *http.Request) (repo.ThemingSettings, error) {
		active, err := ctrl.repo.Themes.GetActive(r.Context())
		return repo.ThemingSettings{Active: active}, err
	}

	return adapters.Command(fn, http.StatusOK)
}

// HandleThemingActiveSet godoc
//
//	@Summary	Set Active Theme
//	@Tags		Theming
//	@Produce	json
//	@Param		payload	body		repo.ThemingSettings	true	"Active theme pointer"
//	@Success	200		{object}	repo.ThemingSettings
//	@Router		/v1/theming/active [Put]
//	@Security	Bearer
func (ctrl *V1Controller) HandleThemingActiveSet() errchain.HandlerFunc {
	fn := func(r *http.Request, body repo.ThemingSettings) (repo.ThemingSettings, error) {
		err := ctrl.repo.Themes.SetActive(r.Context(), body.Active)
		return body, mapThemeError(err)
	}

	return adapters.Action(fn, http.StatusOK)
}

// HandleThemeAssetUpload godoc
//
//	@Summary	Upload Theme Branding Asset
//	@Tags		Theming
//	@Accept		multipart/form-data
//	@Produce	json
//	@Param		id		path		string	true	"Theme ID"
//	@Param		kind	path		string	true	"Asset kind"	Enums(nav-logo, sidebar-logo, login-icon)
//	@Param		file	formData	file	true	"Image file"
//	@Success	200		{object}	repo.ThemeOut
//	@Router		/v1/themes/{id}/assets/{kind} [Post]
//	@Security	Bearer
func (ctrl *V1Controller) HandleThemeAssetUpload() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := ctrl.routeID(r)
		if err != nil {
			return err
		}

		kind, err := themeAssetKind(r)
		if err != nil {
			return err
		}

		if err := r.ParseMultipartForm(ctrl.maxUploadSize << 20); err != nil {
			log.Err(err).Msg("failed to parse multipart form")
			return validate.NewRequestError(errors.New("failed to parse multipart form"), http.StatusBadRequest)
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			return validate.NewRequestError(errors.New("file is required"), http.StatusUnprocessableEntity)
		}
		defer func() { _ = file.Close() }()

		filename := sanitizeAttachmentName(header.Filename)
		ext := strings.ToLower(filepath.Ext(filename))
		if !slices.Contains(themeAssetExtensions, ext) {
			return validate.NewRequestError(errors.New("unsupported image type"), http.StatusUnprocessableEntity)
		}

		out, err := ctrl.repo.Themes.SetAsset(r.Context(), id, kind, filename, file)
		if err != nil {
			log.Err(err).Msg("failed to store theme asset")
			return mapThemeError(err)
		}

		return server.JSON(w, http.StatusOK, out)
	}
}

// HandleThemeAssetDelete godoc
//
//	@Summary	Delete Theme Branding Asset
//	@Tags		Theming
//	@Produce	json
//	@Param		id		path		string	true	"Theme ID"
//	@Param		kind	path		string	true	"Asset kind"	Enums(nav-logo, sidebar-logo, login-icon)
//	@Success	200		{object}	repo.ThemeOut
//	@Router		/v1/themes/{id}/assets/{kind} [Delete]
//	@Security	Bearer
func (ctrl *V1Controller) HandleThemeAssetDelete() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := ctrl.routeID(r)
		if err != nil {
			return err
		}

		kind, err := themeAssetKind(r)
		if err != nil {
			return err
		}

		out, err := ctrl.repo.Themes.DeleteAsset(r.Context(), id, kind)
		if err != nil {
			return mapThemeError(err)
		}

		return server.JSON(w, http.StatusOK, out)
	}
}

// HandleThemeAssetGet godoc
//
//	@Summary	Get Theme Branding Asset (editor preview)
//	@Tags		Theming
//	@Produce	application/octet-stream
//	@Param		id		path	string	true	"Theme ID"
//	@Param		kind	path	string	true	"Asset kind"	Enums(nav-logo, sidebar-logo, login-icon)
//	@Success	200		{file}	file
//	@Router		/v1/themes/{id}/assets/{kind} [Get]
//	@Security	Bearer
func (ctrl *V1Controller) HandleThemeAssetGet() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := ctrl.routeID(r)
		if err != nil {
			return err
		}

		kind, err := themeAssetKind(r)
		if err != nil {
			return err
		}

		path, err := ctrl.repo.Themes.AssetPath(r.Context(), id, kind)
		if err != nil {
			return mapThemeError(err)
		}

		return ctrl.serveThemeAsset(w, r, path)
	}
}

// HandleThemingActiveAssetGet godoc
//
//	@Summary	Get Active Theme Branding Asset
//	@Description	Serves a branding image of the site-wide active theme. Unauthenticated so the login page can use it.
//	@Tags		Theming
//	@Produce	application/octet-stream
//	@Param		kind	path	string	true	"Asset kind"	Enums(nav-logo, sidebar-logo, login-icon)
//	@Success	200		{file}	file
//	@Router		/v1/theming/assets/{kind} [Get]
func (ctrl *V1Controller) HandleThemingActiveAssetGet() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		kind, err := themeAssetKind(r)
		if err != nil {
			return err
		}

		path, err := ctrl.repo.Themes.ActiveAssetPath(r.Context(), kind)
		if err != nil {
			return mapThemeError(err)
		}

		return ctrl.serveThemeAsset(w, r, path)
	}
}

// serveThemeAsset streams a stored branding image. SVGs are served with a
// sandboxing CSP so embedded scripts can never execute even if the file is
// opened directly.
func (ctrl *V1Controller) serveThemeAsset(w http.ResponseWriter, r *http.Request, relativePath string) error {
	if relativePath == "" {
		return validate.NewRequestError(errors.New("no asset uploaded"), http.StatusNotFound)
	}

	bucket, err := blob.OpenBucket(r.Context(), ctrl.repo.Themes.ConnString())
	if err != nil {
		log.Err(err).Msg("failed to open bucket")
		return validate.NewRequestError(err, http.StatusInternalServerError)
	}
	defer func() {
		if err := bucket.Close(); err != nil {
			log.Err(err).Msg("failed to close bucket")
		}
	}()

	file, err := bucket.NewReader(r.Context(), ctrl.repo.Themes.FullAssetPath(relativePath), nil)
	if err != nil {
		log.Err(err).Msg("failed to open theme asset")
		return validate.NewRequestError(err, http.StatusInternalServerError)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Err(err).Msg("failed to close theme asset")
		}
	}()

	name := filepath.Base(relativePath)
	if contentType, ok := themeAssetContentTypes[strings.ToLower(filepath.Ext(name))]; ok {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox;")
	// Assets are immutable per upload; the frontend busts caches with a
	// ?v=<updated_at> query parameter when a slot is replaced.
	w.Header().Set("Cache-Control", "public, max-age=86400")

	http.ServeContent(w, r, name, file.ModTime(), file)
	return nil
}
