package v1

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/hay-kot/httpkit/errchain"
	"github.com/rs/zerolog/log"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
	"github.com/sysadminsmedia/homebox/backend/internal/web/adapters"
	"github.com/sysadminsmedia/homebox/backend/pkgs/hasher"
	"gocloud.dev/blob"
)

// HandlePublicAttachmentGet godoc
//
//	@Summary		Get a thumbnail via a signed public URL
//	@Description	Serves item thumbnails to unauthenticated consumers (e.g. external search UIs) when public image URLs are enabled. The sig parameter is an HMAC over the attachment ID.
//	@Tags			Entities Attachments
//	@Produce		application/octet-stream
//	@Param			attachment_id	path	string	true	"Attachment ID"
//	@Param			sig				query	string	true	"URL signature"
//	@Success		200
//	@Failure		404
//	@Router			/v1/public/attachments/{attachment_id} [GET]
func (ctrl *V1Controller) HandlePublicAttachmentGet() errchain.HandlerFunc {
	notFound := func() error {
		// Disabled, unsigned, tampered, and missing all look identical:
		// this resource does not exist for the caller.
		return validate.NewRequestError(errors.New("not found"), http.StatusNotFound)
	}

	return func(w http.ResponseWriter, r *http.Request) error {
		if !ctrl.runtime().Algolia.PublicImageURLs {
			return notFound()
		}

		attachmentID, err := adapters.RouteUUID(r, "attachment_id")
		if err != nil {
			return notFound()
		}
		if !hasher.VerifyPublicAttachmentSig(attachmentID, r.URL.Query().Get("sig")) {
			return notFound()
		}

		doc, err := ctrl.repo.Attachments.GetPublicThumbnail(r.Context(), attachmentID)
		if err != nil {
			return notFound()
		}

		bucket, err := blob.OpenBucket(r.Context(), ctrl.repo.Attachments.GetConnString())
		if err != nil {
			log.Err(err).Msg("public attachment: failed to open bucket")
			return validate.NewRequestError(err, http.StatusInternalServerError)
		}
		defer func() {
			if err := bucket.Close(); err != nil {
				log.Err(err).Msg("public attachment: failed to close bucket")
			}
		}()

		file, err := bucket.NewReader(r.Context(), ctrl.repo.Attachments.GetFullPath(doc.Path), nil)
		if err != nil {
			log.Err(err).Msg("public attachment: failed to open file")
			return validate.NewRequestError(err, http.StatusInternalServerError)
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Err(err).Msg("public attachment: failed to close file")
			}
		}()

		w.Header().Set("Content-Disposition", "inline; filename*=UTF-8''"+url.QueryEscape(doc.Title))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Download-Options", "noopen")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self'; style-src 'unsafe-inline'; sandbox;")
		// Signed URLs are stable, so downstream caches (and Algolia search
		// UIs re-rendering results) can hold thumbnails for a day.
		w.Header().Set("Cache-Control", "public, max-age=86400")

		http.ServeContent(w, r, doc.Title, doc.CreatedAt, file)
		return nil
	}
}
