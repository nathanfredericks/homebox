// Package config provides the configuration for the application.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/ardanlabs/conf/v3"
)

// redactedValue is the sentinel substituted for any sensitive field when the
// configuration is serialized (e.g. via Print). It must not match any plausible
// real value.
const redactedValue = "[REDACTED]"

// redactURLUserinfo returns raw with any password component of an embedded
// userinfo section replaced by the redaction sentinel. The username is left
// visible so operators can still recognize which account is configured. Inputs
// that are not parseable URLs, or that contain no password, are returned
// unchanged.
func redactURLUserinfo(raw string) string {
	if raw == "" {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	if _, hasPassword := u.User.Password(); !hasPassword {
		return raw
	}
	// Plain "REDACTED" is used here (rather than redactedValue) so it survives
	// URL percent-encoding without becoming "%5BREDACTED%5D".
	u.User = url.UserPassword(u.User.Username(), "REDACTED")
	return u.String()
}

const (
	ModeDevelopment = "development"
	ModeProduction  = "production"
)

type Config struct {
	conf.Version
	Mode       string         `yaml:"mode"       conf:"default:development"` // development or production
	Web        WebConfig      `yaml:"web"`
	Storage    Storage        `yaml:"storage"`
	Database   Database       `yaml:"database"`
	Log        LoggerConf     `yaml:"logger"`
	Mailer     MailerConf     `yaml:"mailer"`
	Demo       bool           `yaml:"demo"`
	Debug      DebugConf      `yaml:"debug"`
	Options    Options        `yaml:"options"`
	OIDC       OIDCConf       `yaml:"oidc"`
	LabelMaker LabelMakerConf `yaml:"labelmaker"`
	Thumbnail  Thumbnail      `yaml:"thumbnail"`
	Barcode    BarcodeAPIConf `yaml:"barcode"`
	Otel       OTelConfig     `yaml:"otel"`
	Auth       AuthConfig     `yaml:"auth"`
	Notifier   NotifierConf   `yaml:"notifier"`
	Algolia    AlgoliaConf    `yaml:"algolia"`
}

type Options struct {
	AllowRegistration    bool   `json:"allowRegistration"    yaml:"disable_registration"    conf:"default:true"`
	AutoIncrementAssetID bool   `json:"autoIncrementAssetId" yaml:"auto_increment_asset_id" conf:"default:true"`
	CurrencyConfig       string `json:"currencyConfig"       yaml:"currencies"`
	GithubReleaseCheck   bool   `json:"githubReleaseCheck"   yaml:"check_github_release"    conf:"default:true"`
	AllowAnalytics       bool   `json:"allowAnalytics"       yaml:"allow_analytics"         conf:"default:false"`
	AllowLocalLogin      bool   `json:"allowLocalLogin"      yaml:"allow_local_login"       conf:"default:true"`
	TrustProxy           bool   `json:"trustProxy"           yaml:"trust_proxy"             conf:"default:false"`
	Hostname             string `json:"hostname"             yaml:"hostname"`
}

type Thumbnail struct {
	Enabled bool `json:"enabled" yaml:"enabled" conf:"default:true"`
	Width   int  `json:"width"   yaml:"width"   conf:"default:500"`
	Height  int  `json:"height"  yaml:"height"  conf:"default:500"`
}

type DebugConf struct {
	Enabled bool   `yaml:"enabled" conf:"default:false"`
	Port    string `yaml:"port"    conf:"default:4000"`
}

type WebConfig struct {
	Port string `yaml:"port" conf:"default:7745"`
	Host string `yaml:"host"`
	// MaxUploadSize is the body cap (in MB) applied to ordinary upload
	// endpoints (attachments, item imports, etc.). Defaults to 10 MB.
	MaxUploadSize int64 `yaml:"max_file_upload" conf:"default:10"`
	// MaxImportSize is the body cap (in MB) for collection-restore uploads
	// (POST /v1/group/import). Set independently because a full collection
	// backup including attachments can be much larger than a single asset
	// upload. Defaults to 1 GB.
	MaxImportSize int64         `yaml:"max_import_upload" conf:"default:1024"`
	ReadTimeout   time.Duration `yaml:"read_timeout"      conf:"default:10s"`
	WriteTimeout  time.Duration `yaml:"write_timeout"     conf:"default:10s"`
	IdleTimeout   time.Duration `yaml:"idle_timeout"      conf:"default:30s"`
}

type LabelMakerConf struct {
	Width                 int64          `json:"width"                 yaml:"width"                 conf:"default:526"`
	Height                int64          `json:"height"                yaml:"height"                conf:"default:200"`
	Padding               int64          `json:"padding"               yaml:"padding"               conf:"default:32"`
	Margin                int64          `json:"margin"                yaml:"margin"                conf:"default:32"`
	FontSize              float64        `json:"fontSize"              yaml:"font_size"             conf:"default:32.0"`
	PrintCommand          *string        `json:"printCommand"          yaml:"string"`
	AdditionalInformation *string        `json:"additionalInformation" yaml:"string"`
	DynamicLength         bool           `json:"dynamicLength"         yaml:"bool"                  conf:"default:true"`
	LabelServiceUrl       *string        `json:"labelServiceUrl"       yaml:"label_service_url"`
	LabelServiceTimeout   *time.Duration `json:"labelServiceTimeout"   yaml:"label_service_timeout"`
	RegularFontPath       *string        `json:"regularFontPath"       yaml:"regular_font_path"`
	BoldFontPath          *string        `json:"boldFontPath"          yaml:"bold_font_path"`
}

type OIDCConf struct {
	Enabled            bool          `yaml:"enabled"              conf:"default:false"`
	IssuerURL          string        `yaml:"issuer_url"`
	ClientID           string        `yaml:"client_id"`
	ClientSecret       string        `yaml:"client_secret"`
	Scope              string        `yaml:"scope"                conf:"default:openid profile email"`
	AllowedGroups      string        `yaml:"allowed_groups"`
	AutoRedirect       bool          `yaml:"auto_redirect"        conf:"default:false"`
	VerifyEmail        bool          `yaml:"verify_email"         conf:"default:false"`
	GroupClaim         string        `yaml:"group_claim"          conf:"default:groups"`
	EmailClaim         string        `yaml:"email_claim"          conf:"default:email"`
	NameClaim          string        `yaml:"name_claim"           conf:"default:name"`
	EmailVerifiedClaim string        `yaml:"email_verified_claim" conf:"default:email_verified"`
	ButtonText         string        `yaml:"button_text"          conf:"default:Sign in with OIDC"`
	StateExpiry        time.Duration `yaml:"state_expiry"         conf:"default:10m"`
	RequestTimeout     time.Duration `yaml:"request_timeout"      conf:"default:30s"`
}

func (c OIDCConf) MarshalJSON() ([]byte, error) {
	type alias OIDCConf
	a := alias(c)
	if a.ClientSecret != "" {
		a.ClientSecret = redactedValue
	}
	return json.Marshal(a)
}

type BarcodeAPIConf struct {
	TokenBarcodespider   string `json:"tokenBarcodespider"   yaml:"token_barcodespider"`
	OpenFoodFactsContact string `json:"openFoodFactsContact" yaml:"openfoodfacts_contact"`
}

func (c BarcodeAPIConf) MarshalJSON() ([]byte, error) {
	type alias BarcodeAPIConf
	a := alias(c)
	if a.TokenBarcodespider != "" {
		a.TokenBarcodespider = redactedValue
	}
	return json.Marshal(a)
}

// AlgoliaConf configures pushing item records to an Algolia search index.
// Every field can be overridden at runtime through the site settings UI; the
// values here only provide the environment/default layer.
type AlgoliaConf struct {
	Enabled     bool   `json:"enabled"     yaml:"enabled"       conf:"default:false"`
	AppID       string `json:"appId"       yaml:"app_id"`
	AdminAPIKey string `json:"adminApiKey" yaml:"admin_api_key" conf:"mask"`
	IndexName   string `json:"indexName"   yaml:"index_name"    conf:"default:homebox-items"`
	// Fields is a comma-separated allowlist of record fields to push. Empty
	// means every field. objectID and groupId are always included.
	Fields string `json:"fields" yaml:"fields"`
	// PublicImageURLs includes an unauthenticated HMAC-signed thumbnail URL in
	// each record so external search UIs can render item images.
	PublicImageURLs bool `json:"publicImageUrls" yaml:"public_image_urls" conf:"default:false"`
	// PublicBaseURL is the externally reachable base URL used to build
	// thumbnail links. Falls back to options.hostname when empty.
	PublicBaseURL string `json:"publicBaseUrl" yaml:"public_base_url"`
	// ReindexInterval is a Go duration string ("24h"); kept as a string so it
	// round-trips through the settings API and UI without nanosecond math.
	ReindexInterval string `json:"reindexInterval" yaml:"reindex_interval" conf:"default:24h"`
}

func (c AlgoliaConf) MarshalJSON() ([]byte, error) {
	type alias AlgoliaConf
	a := alias(c)
	if a.AdminAPIKey != "" {
		a.AdminAPIKey = redactedValue
	}
	return json.Marshal(a)
}

type AuthConfig struct {
	RateLimit AuthRateLimit `yaml:"rate_limit"`
	// APIKeyPepper is a server-side secret HMAC-keyed into stored API key hashes.
	// Holding it outside the database means a DB-only leak yields no usable hashes.
	// Must stay stable across restarts — rotating it invalidates every issued key.
	// Generate with `openssl rand -base64 48`.
	APIKeyPepper string `yaml:"api_key_pepper" conf:"mask"`
}

func (c AuthConfig) MarshalJSON() ([]byte, error) {
	type alias AuthConfig
	a := alias(c)
	if a.APIKeyPepper != "" {
		a.APIKeyPepper = redactedValue
	}
	return json.Marshal(a)
}

type AuthRateLimit struct {
	Enabled     bool          `yaml:"enabled"      conf:"default:true"`
	Window      time.Duration `yaml:"window"       conf:"default:1m"`
	MaxAttempts int           `yaml:"max_attempts" conf:"default:5"`
	BaseBackoff time.Duration `yaml:"base_backoff" conf:"default:10s"`
	MaxBackoff  time.Duration `yaml:"max_backoff"  conf:"default:5m"`
}

// New parses the CLI/Config file and returns a Config struct. If the file argument is an empty string, the
// file is not read. If the file is not empty, the file is read and the Config struct is returned.
func New(buildstr string, description string) (*Config, error) {
	var cfg Config
	const prefix = "HBOX"

	cfg.Version = conf.Version{
		Build: buildstr,
		Desc:  description,
	}

	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			os.Exit(0)
		}
		return &cfg, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// Defaults returns a Config populated only from the struct `conf` default
// tags: the parse uses a prefix that matches no environment variables, so
// HBOX_* values are ignored. The admin settings service uses this as the base
// for its runtime-changeable sections, making the database the only override;
// bootstrap configuration keeps reading the environment via New.
func Defaults() (*Config, error) {
	var cfg Config
	if _, err := conf.Parse("HBOXDEFAULTSONLY", &cfg); err != nil {
		return nil, fmt.Errorf("parsing default config: %w", err)
	}
	return &cfg, nil
}

// Print prints the configuration to stdout as an indented JSON document.
// Sensitive fields (secrets, tokens, passwords, embedded URL credentials) are
// redacted via each sub-struct's MarshalJSON. Useful for debugging operator
// configuration without leaking credentials to logs.
func (c *Config) Print() {
	res, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(res))
}
