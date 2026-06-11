// Package settings resolves runtime-changeable configuration by layering
// database overrides (edited through the admin settings UI) on top of the
// environment/default configuration parsed at startup. Bootstrap-critical
// config (database, web server, storage, auth, OIDC, ...) never goes through
// this package and remains environment-only.
package settings

import (
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
)

// Resolved is the effective runtime configuration: the startup config with
// any database overrides applied. Secrets are redacted on marshal by the
// config sub-structs' MarshalJSON methods, so a Resolved value is safe to
// serialize for the admin API.
type Resolved struct {
	Options    config.Options        `json:"options"`
	Thumbnail  config.Thumbnail      `json:"thumbnail"`
	Barcode    config.BarcodeAPIConf `json:"barcode"`
	Mailer     config.MailerConf     `json:"mailer"`
	LabelMaker config.LabelMakerConf `json:"labelmaker"`
	Notifier   config.NotifierConf   `json:"notifier"`
	Algolia    config.AlgoliaConf    `json:"algolia"`
}

// Section names double as database row keys and API path segments.
const (
	SectionOptions    = "options"
	SectionThumbnail  = "thumbnail"
	SectionBarcode    = "barcode"
	SectionMailer     = "mailer"
	SectionLabelMaker = "labelmaker"
	SectionNotifier   = "notifier"
	SectionAlgolia    = "algolia"
)

// SectionNames lists every valid section in UI display order.
var SectionNames = []string{
	SectionOptions,
	SectionThumbnail,
	SectionMailer,
	SectionBarcode,
	SectionLabelMaker,
	SectionNotifier,
	SectionAlgolia,
}

// sectionPtr returns the pointer to the named section within r, used both to
// apply database overrides and to validate incoming payloads.
func sectionPtr(r *Resolved, name string) any {
	switch name {
	case SectionOptions:
		return &r.Options
	case SectionThumbnail:
		return &r.Thumbnail
	case SectionBarcode:
		return &r.Barcode
	case SectionMailer:
		return &r.Mailer
	case SectionLabelMaker:
		return &r.LabelMaker
	case SectionNotifier:
		return &r.Notifier
	case SectionAlgolia:
		return &r.Algolia
	default:
		return nil
	}
}

// sectionSecrets maps a section to the JSON keys of its write-only secret
// fields. Reads return the redaction sentinel; an update echoing the sentinel
// back keeps the currently effective value.
var sectionSecrets = map[string][]string{
	SectionBarcode: {"tokenBarcodespider"},
	SectionMailer:  {"password"},
	SectionAlgolia: {"adminApiKey"},
}

// currentSecret returns the currently effective value for a section's secret
// field, used to substitute the sentinel on update.
func currentSecret(r *Resolved, section, jsonKey string) string {
	switch section + "/" + jsonKey {
	case SectionBarcode + "/tokenBarcodespider":
		return r.Barcode.TokenBarcodespider
	case SectionMailer + "/password":
		return r.Mailer.Password
	case SectionAlgolia + "/adminApiKey":
		return r.Algolia.AdminAPIKey
	default:
		return ""
	}
}

// sectionEnvOnly maps a section to JSON keys that must not be overridden from
// the database even though they live in the same config struct: they are
// either read once at startup or security-sensitive trust decisions.
var sectionEnvOnly = map[string][]string{
	SectionOptions: {"currencyConfig", "allowAnalytics", "trustProxy"},
}
