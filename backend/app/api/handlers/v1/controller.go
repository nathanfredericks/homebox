// Package v1 provides the API handlers for version 1 of the API.
package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hay-kot/httpkit/errchain"
	"github.com/hay-kot/httpkit/server"
	"github.com/rs/zerolog/log"
	"github.com/sysadminsmedia/homebox/backend/app/api/providers"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/settings"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/schema"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"

	"github.com/olahol/melody"
)

type Results[T any] struct {
	Items []T `json:"items"`
}

func WrapResults[T any](items []T) Results[T] {
	return Results[T]{Items: items}
}

type Wrapped struct {
	Item interface{} `json:"item"`
}

func Wrap(v any) Wrapped {
	return Wrapped{Item: v}
}

func WithMaxUploadSize(maxUploadSize int64) func(*V1Controller) {
	return func(ctrl *V1Controller) {
		ctrl.maxUploadSize = maxUploadSize
	}
}

func WithMaxImportSize(maxImportSize int64) func(*V1Controller) {
	return func(ctrl *V1Controller) {
		ctrl.maxImportSize = maxImportSize
	}
}

func WithDemoStatus(demoStatus bool) func(*V1Controller) {
	return func(ctrl *V1Controller) {
		ctrl.isDemo = demoStatus
	}
}

func WithRegistration(allowRegistration bool) func(*V1Controller) {
	return func(ctrl *V1Controller) {
		ctrl.allowRegistration = allowRegistration
	}
}

func WithSecureCookies(secure bool) func(*V1Controller) {
	return func(ctrl *V1Controller) {
		ctrl.cookieSecure = secure
	}
}

func WithURL(url string) func(*V1Controller) {
	return func(ctrl *V1Controller) {
		ctrl.url = url
	}
}

// WithSettings wires the site settings service so handlers read
// runtime-changeable configuration per request instead of freezing the
// startup config.
func WithSettings(s *settings.Service) func(*V1Controller) {
	return func(ctrl *V1Controller) {
		ctrl.settings = s
	}
}

// WithAlgoliaReindex registers the callback behind POST
// /admin/settings/algolia/reindex. A nil callback keeps the endpoint
// returning 404.
func WithAlgoliaReindex(fn func()) func(*V1Controller) {
	return func(ctrl *V1Controller) {
		ctrl.algoliaReindex = fn
	}
}

type V1Controller struct {
	cookieSecure      bool
	repo              *repo.AllRepos
	svc               *services.AllServices
	maxUploadSize     int64
	maxImportSize     int64
	isDemo            bool
	allowRegistration bool
	setupComplete     atomic.Bool
	bus               *eventbus.EventBus
	url               string
	config            *config.Config
	settings          *settings.Service
	algoliaReindex    func()
	oidcProvider      *providers.OIDCProvider
}

// hbURL resolves the instance base URL for a request from runtime options.
func (ctrl *V1Controller) hbURL(r *http.Request) string {
	opts := ctrl.runtime().Options
	return GetHBURL(r, &opts, ctrl.url)
}

// secureBaseURL is hbURL without the Referer fallback, for URLs embedded in
// emails where a forged header must not be able to poison the link.
func (ctrl *V1Controller) secureBaseURL(r *http.Request) string {
	opts := ctrl.runtime().Options
	return SecureBaseURL(r, &opts)
}

// runtime returns the effective runtime-changeable configuration: the site
// settings snapshot when the service is wired, otherwise the startup config.
func (ctrl *V1Controller) runtime() settings.Resolved {
	if ctrl.settings != nil {
		return ctrl.settings.Get()
	}
	return settings.Resolved{
		Options:    ctrl.config.Options,
		Thumbnail:  ctrl.config.Thumbnail,
		Barcode:    ctrl.config.Barcode,
		Mailer:     ctrl.config.Mailer,
		LabelMaker: ctrl.config.LabelMaker,
		Notifier:   ctrl.config.Notifier,
		Algolia:    ctrl.config.Algolia,
	}
}

type (
	ReadyFunc func() bool

	Build struct {
		Version   string `json:"version"`
		Commit    string `json:"commit"`
		BuildTime string `json:"buildTime"`
	}

	APISummary struct {
		Healthy           bool            `json:"health"`
		Versions          []string        `json:"versions"`
		Title             string          `json:"title"`
		Message           string          `json:"message"`
		Build             Build           `json:"build"`
		Latest            services.Latest `json:"latest"`
		Demo              bool            `json:"demo"`
		AllowRegistration bool            `json:"allowRegistration"`
		// Setup is true while no user exists; the frontend shows the
		// first-time setup flow instead of the login form.
		Setup         bool            `json:"setup"`
		LabelPrinting bool            `json:"labelPrinting"`
		OIDC          OIDCStatus      `json:"oidc"`
		Telemetry     TelemetryStatus `json:"telemetry"`
		Theming       ThemingStatus   `json:"theming"`
	}

	// ThemingStatus describes the site-wide active theme so pre-auth pages
	// (login) can render colors, fonts and branding. Colors/fonts/branding
	// are only populated for custom themes; built-ins live in the frontend.
	ThemingStatus struct {
		Active   string                `json:"active"`
		Name     string                `json:"name,omitempty"`
		Colors   map[string]string     `json:"colors,omitempty"`
		Radius   string                `json:"radius,omitempty"`
		FontSans string                `json:"fontSans,omitempty"`
		FontMono string                `json:"fontMono,omitempty"`
		Branding *schema.ThemeBranding `json:"branding,omitempty"`
		Assets   repo.ThemeAssets      `json:"assets"`
		// Version changes whenever the active theme row changes; the
		// frontend appends it to asset URLs as a cache buster.
		Version int64 `json:"version,omitempty"`
	}

	OIDCStatus struct {
		Enabled      bool   `json:"enabled"`
		ButtonText   string `json:"buttonText,omitempty"`
		AutoRedirect bool   `json:"autoRedirect,omitempty"`
		AllowLocal   bool   `json:"allowLocal"`
	}

	TelemetryStatus struct {
		Enabled bool `json:"enabled"`
	}
)

func NewControllerV1(svc *services.AllServices, repos *repo.AllRepos, bus *eventbus.EventBus, config *config.Config, options ...func(*V1Controller)) *V1Controller {
	ctrl := &V1Controller{
		repo:              repos,
		svc:               svc,
		allowRegistration: true,
		bus:               bus,
		config:            config,
	}

	for _, opt := range options {
		opt(ctrl)
	}

	ctrl.initOIDCProvider()

	return ctrl
}

func (ctrl *V1Controller) initOIDCProvider() {
	if ctrl.config.OIDC.Enabled {
		oidcProvider, err := providers.NewOIDCProvider(ctrl.svc.User, &ctrl.config.OIDC, &ctrl.config.Options, ctrl.cookieSecure)
		if err != nil {
			log.Err(err).Msg("failed to initialize OIDC provider at startup")
		} else {
			ctrl.oidcProvider = oidcProvider
			log.Info().Msg("OIDC provider initialized successfully at startup")
		}
	}
}

// HandleBase godoc
//
//	@Summary	Application Info
//	@Tags		Base
//	@Produce	json
//	@Success	200	{object}	APISummary
//	@Router		/v1/status [GET]
func (ctrl *V1Controller) HandleBase(ready ReadyFunc, build Build) errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		// Setup mode ends permanently once the first user exists, so cache
		// the false result and skip the count query afterwards.
		setup := false
		if !ctrl.setupComplete.Load() {
			count, err := ctrl.repo.Users.Count(r.Context())
			if err != nil {
				log.Err(err).Msg("failed to count users for setup status")
			} else if count == 0 {
				setup = true
			} else {
				ctrl.setupComplete.Store(true)
			}
		}

		theming := ctrl.themingStatus(r.Context())
		title := "Homebox"
		if theming.Branding != nil && theming.Branding.AppName != "" {
			title = theming.Branding.AppName
		}

		rt := ctrl.runtime()
		return server.JSON(w, http.StatusOK, APISummary{
			Healthy:           ready(),
			Title:             title,
			Message:           "Track, Manage, and Organize your Things",
			Build:             build,
			Latest:            ctrl.svc.BackgroundService.GetLatestVersion(),
			Demo:              ctrl.isDemo,
			AllowRegistration: false,
			Setup:             setup,
			LabelPrinting:     rt.LabelMaker.PrintCommand != nil,
			OIDC: OIDCStatus{
				Enabled:      ctrl.config.OIDC.Enabled,
				ButtonText:   ctrl.config.OIDC.ButtonText,
				AutoRedirect: ctrl.config.OIDC.AutoRedirect,
				AllowLocal:   rt.Options.AllowLocalLogin,
			},
			Telemetry: TelemetryStatus{
				Enabled: ctrl.config.Otel.Enabled,
			},
			Theming: theming,
		})
	}
}

// themingStatus resolves the active theme for the public status payload.
// Failures degrade to the default built-in theme rather than failing /status.
func (ctrl *V1Controller) themingStatus(ctx context.Context) ThemingStatus {
	active, theme, err := ctrl.repo.Themes.GetActiveTheme(ctx)
	if err != nil {
		log.Err(err).Msg("failed to resolve active theme for status")
		return ThemingStatus{Active: repo.DefaultActiveTheme}
	}

	status := ThemingStatus{Active: active}
	if theme != nil {
		branding := theme.Branding
		status.Name = theme.Name
		status.Colors = theme.Colors
		status.Radius = theme.Radius
		status.FontSans = theme.FontSans
		status.FontMono = theme.FontMono
		status.Branding = &branding
		status.Assets = theme.Assets
		status.Version = theme.UpdatedAt.Unix()
	}
	return status
}

// HandleCurrency godoc
//
//	@Summary	Currency
//	@Tags		Base
//	@Produce	json
//	@Success	200	{object}	currencies.Currency
//	@Router		/v1/currency [GET]
func (ctrl *V1Controller) HandleCurrency() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		// Set Cache for 10 Minutes
		w.Header().Set("Cache-Control", "max-age=600")

		return server.JSON(w, http.StatusOK, ctrl.svc.Currencies.Slice())
	}
}

func (ctrl *V1Controller) HandleCacheWS() errchain.HandlerFunc {
	type eventMsg struct {
		Event string `json:"event"`
	}

	m := melody.New()
	m.Upgrader.Subprotocols = []string{"hb-auth"}

	m.HandleConnect(func(s *melody.Session) {
		auth := services.NewContext(s.Request.Context())
		s.Set("gid", auth.GID)
	})

	factory := func(e string) func(data any) {
		return func(data any) {
			eventData, ok := data.(eventbus.GroupMutationEvent)
			if !ok {
				log.Log().Msgf("invalid event data: %v", data)
				return
			}

			msg := &eventMsg{Event: e}

			jsonBytes, err := json.Marshal(msg)
			if err != nil {
				log.Log().Msgf("error marshling event data %v: %v", data, err)
				return
			}

			_ = m.BroadcastFilter(jsonBytes, func(s *melody.Session) bool {
				groupIDStr, ok := s.Get("gid")
				if !ok {
					return false
				}

				GID := groupIDStr.(uuid.UUID)
				return GID == eventData.GID
			})
		}
	}

	ctrl.bus.Subscribe(eventbus.EventTagMutation, factory("tag.mutation"))
	ctrl.bus.Subscribe(eventbus.EventEntityMutation, factory("entity.mutation"))
	ctrl.bus.Subscribe(eventbus.EventUserMutation, factory("user.mutation"))
	ctrl.bus.Subscribe(eventbus.EventExportMutation, factory("export.mutation"))
	ctrl.bus.Subscribe(eventbus.EventImportMutation, factory("import.mutation"))

	// Persistent asynchronous ticker that keeps all websocket connections alive with periodic pings.
	go func() {
		const interval = 10 * time.Second

		ping := time.NewTicker(interval)
		defer ping.Stop()

		for range ping.C {
			msg := &eventMsg{Event: "ping"}

			pingBytes, err := json.Marshal(msg)
			if err != nil {
				log.Log().Msgf("error marshaling ping: %v", err)
			} else {
				_ = m.Broadcast(pingBytes)
			}
		}
	}()

	return func(w http.ResponseWriter, r *http.Request) error {
		return m.HandleRequest(w, r)
	}
}
