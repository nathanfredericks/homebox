// Package algolia keeps an Algolia search index in sync with the inventory:
// debounced incremental pushes on entity mutations plus periodic and
// on-demand full reindexes. Indexing failures are logged and never propagate
// into user-facing operations.
package algolia

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	"github.com/sysadminsmedia/homebox/backend/pkgs/hasher"
)

// RecordFields lists every optional record field, mirroring the itemSummary
// shape of the homebox-items Lambda. The field allowlist in the Algolia
// settings is validated against this list; objectID and groupId are always
// present and not listed.
var RecordFields = []string{
	"assetId",
	"name",
	"description",
	"quantity",
	"insured",
	"archived",
	"purchasePrice",
	"location",
	"tags",
	"soldTime",
	"thumbnailUrl",
	"createdAt",
	"updatedAt",
	"lifetimeWarranty",
	"manufacturer",
	"modelNumber",
	"serialNumber",
	"purchaseFrom",
	"purchaseTime",
	"soldTo",
	"soldPrice",
	"soldNotes",
	"notes",
	"warrantyDetails",
	"warrantyExpires",
}

// parseFieldAllowlist turns the comma-separated setting into a lookup set.
// nil means "all fields"; unknown names are ignored.
func parseFieldAllowlist(csv string) map[string]bool {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return nil
	}
	known := make(map[string]string, len(RecordFields))
	for _, f := range RecordFields {
		known[strings.ToLower(f)] = f
	}
	out := map[string]bool{}
	for _, raw := range strings.Split(csv, ",") {
		if f, ok := known[strings.ToLower(strings.TrimSpace(raw))]; ok {
			out[f] = true
		}
	}
	return out
}

// dateOrNil renders a date-only value as its string form, nil when unset, so
// records don't carry Go zero-time noise (mirrors the Lambda's soldTime
// handling).
func dateOrNil(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.Format("2006-01-02")
}

// buildRecord flattens one item into an Algolia record. paths maps entity ID
// to its full ancestor path; the item's location is its parent's path.
// publicBase is the scheme-qualified external base URL, empty when public
// image URLs are disabled.
func buildRecord(e repo.EntityOut, gid uuid.UUID, paths map[uuid.UUID]string, allow map[string]bool, publicBase string) map[string]any {
	var location any
	if e.Parent != nil {
		if p, ok := paths[e.Parent.ID]; ok {
			location = p
		} else {
			location = e.Parent.Name
		}
	}

	var thumbnailURL any
	if publicBase != "" && e.ThumbnailId != nil {
		id := *e.ThumbnailId
		thumbnailURL = publicBase + "/api/v1/public/attachments/" + id.String() + "?sig=" + hasher.SignPublicAttachmentID(id)
	}

	tags := e.Tags
	if tags == nil {
		tags = []repo.TagSummary{}
	}

	full := map[string]any{
		"assetId":          e.AssetID.String(),
		"name":             e.Name,
		"description":      e.Description,
		"quantity":         e.Quantity,
		"insured":          e.Insured,
		"archived":         e.Archived,
		"purchasePrice":    e.PurchasePrice,
		"location":         location,
		"tags":             tags,
		"soldTime":         dateOrNil(e.SoldDate.Time()),
		"thumbnailUrl":     thumbnailURL,
		"createdAt":        e.CreatedAt.Format(time.RFC3339),
		"updatedAt":        e.UpdatedAt.Format(time.RFC3339),
		"lifetimeWarranty": e.LifetimeWarranty,
		"manufacturer":     e.Manufacturer,
		"modelNumber":      e.ModelNumber,
		"serialNumber":     e.SerialNumber,
		"purchaseFrom":     e.PurchaseFrom,
		"purchaseTime":     dateOrNil(e.PurchaseDate.Time()),
		"soldTo":           e.SoldTo,
		"soldPrice":        e.SoldPrice,
		"soldNotes":        e.SoldNotes,
		"notes":            e.Notes,
		"warrantyDetails":  e.WarrantyDetails,
		"warrantyExpires":  dateOrNil(e.WarrantyExpires.Time()),
	}

	rec := map[string]any{
		"objectID": e.ID.String(),
		"groupId":  gid.String(),
	}
	for k, v := range full {
		if allow == nil || allow[k] {
			rec[k] = v
		}
	}
	return rec
}

// publicBaseURL resolves the externally reachable base URL for thumbnail
// links: the explicit setting, else the instance hostname. Returns "" when
// public image URLs are disabled or no base is configured (no request context
// exists during sync, so there is nothing to infer a host from).
func publicBaseURL(conf config.AlgoliaConf, hostname string) string {
	if !conf.PublicImageURLs {
		return ""
	}
	base := strings.TrimSpace(conf.PublicBaseURL)
	if base == "" {
		base = strings.TrimSpace(hostname)
	}
	if base == "" {
		return ""
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "https://" + base
	}
	return strings.TrimSuffix(base, "/")
}
