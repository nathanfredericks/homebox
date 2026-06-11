package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hay-kot/httpkit/errchain"
	httpSwagger "github.com/swaggo/http-swagger/v2" // http-swagger middleware
	"github.com/sysadminsmedia/homebox/backend/app/api/handlers/debughandlers"
	v1 "github.com/sysadminsmedia/homebox/backend/app/api/handlers/v1"
	"github.com/sysadminsmedia/homebox/backend/app/api/providers"
	docs "github.com/sysadminsmedia/homebox/backend/app/api/static/docs"
	"github.com/sysadminsmedia/homebox/backend/internal/core/permissions"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/authroles"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
)

const prefix = "/api"

func (a *app) debugRouter() *http.ServeMux {
	dbg := http.NewServeMux()
	debughandlers.New(dbg)

	return dbg
}

// registerRoutes registers all the routes for the API
func (a *app) mountRoutes(r *chi.Mux, chain *errchain.ErrChain, repos *repo.AllRepos) {
	// Serve doc.json dynamically so the Swagger UI "Base URL" reflects the
	// actual host of the user's instance rather than a hardcoded value.
	r.Get("/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if fwdHost := r.Header.Get("X-Forwarded-Host"); fwdHost != "" {
			host = fwdHost
		}
		spec := *docs.SwaggerInfo
		spec.Host = host
		doc := spec.ReadDoc()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(doc))
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// =========================================================================
	// API Version 1

	v1Ctrl := v1.NewControllerV1(
		a.services,
		a.repos,
		a.bus,
		a.conf,
		v1.WithMaxUploadSize(a.conf.Web.MaxUploadSize),
		v1.WithMaxImportSize(a.conf.Web.MaxImportSize),
		v1.WithRegistration(a.conf.Options.AllowRegistration),
		v1.WithDemoStatus(a.conf.Demo), // Disable Password Change in Demo Mode
		v1.WithURL(fmt.Sprintf("%s:%s", a.conf.Web.Host, a.conf.Web.Port)),
		v1.WithSettings(a.settings),
		v1.WithAlgoliaReindex(a.algolia.FullReindex),
	)

	r.Route(prefix+"/v1", func(r chi.Router) {
		r.Get("/status", chain.ToHandlerFunc(v1Ctrl.HandleBase(func() bool { return true }, v1.Build{
			Version:   version,
			Commit:    commit,
			BuildTime: buildTime,
		})))

		r.Get("/currencies", chain.ToHandlerFunc(v1Ctrl.HandleCurrency()))

		providers := []v1.AuthProvider{
			providers.NewLocalProvider(a.services.User),
		}

		r.Post("/users/register", chain.ToHandlerFunc(v1Ctrl.HandleUserRegistration()))
		r.Post("/users/login", chain.ToHandlerFunc(v1Ctrl.HandleAuthLogin(providers...), a.mwAuthRateLimit))
		r.Post("/users/forgot-password", chain.ToHandlerFunc(v1Ctrl.HandleForgotPassword(), a.mwAuthRateLimit))
		r.Post("/users/reset-password", chain.ToHandlerFunc(v1Ctrl.HandleResetPassword(), a.mwAuthRateLimit))

		if a.conf.OIDC.Enabled {
			r.Get("/users/login/oidc", chain.ToHandlerFunc(v1Ctrl.HandleOIDCLogin(), a.mwAuthRateLimit))
			r.Get("/users/login/oidc/callback", chain.ToHandlerFunc(v1Ctrl.HandleOIDCCallback(), a.mwAuthRateLimit))
		}

		userMW := []errchain.Middleware{
			a.mwAuthToken,
			a.mwTenant,
			a.mwRoles(RoleModeOr, authroles.RoleUser.String()),
		}

		// siteMW guards tenant-independent endpoints (user/role administration,
		// collection creation). mwPermission evaluates site-scoped sections.
		siteMW := []errchain.Middleware{
			a.mwAuthToken,
			a.mwRoles(RoleModeOr, authroles.RoleUser.String()),
		}

		// with appends permission middleware to a base chain without sharing
		// backing arrays between routes.
		with := func(base []errchain.Middleware, extra ...errchain.Middleware) []errchain.Middleware {
			out := make([]errchain.Middleware, 0, len(base)+len(extra))
			out = append(out, base...)
			return append(out, extra...)
		}

		entitySections := []permissions.Section{permissions.SectionItems, permissions.SectionLocations}

		r.Get("/ws/events", chain.ToHandlerFunc(v1Ctrl.HandleCacheWS(), userMW...))

		// User self endpoints (always allowed for the authenticated user)
		r.Get("/users/self", chain.ToHandlerFunc(v1Ctrl.HandleUserSelf(), userMW...))
		r.Put("/users/self", chain.ToHandlerFunc(v1Ctrl.HandleUserSelfUpdate(), userMW...))
		r.Delete("/users/self", chain.ToHandlerFunc(v1Ctrl.HandleUserSelfDelete(), userMW...))
		r.Get("/users/self/settings", chain.ToHandlerFunc(v1Ctrl.HandleUserSelfSettingsGet(), userMW...))
		r.Put("/users/self/settings", chain.ToHandlerFunc(v1Ctrl.HandleUserSelfSettingsUpdate(), userMW...))
		r.Post("/users/logout", chain.ToHandlerFunc(v1Ctrl.HandleAuthLogout(), userMW...))
		r.Get("/users/refresh", chain.ToHandlerFunc(v1Ctrl.HandleAuthRefresh(), userMW...))
		r.Put("/users/self/change-password", chain.ToHandlerFunc(v1Ctrl.HandleUserSelfChangePassword(), userMW...))

		// User API keys (static tokens that authenticate as the owning user)
		r.Get("/users/self/api-keys", chain.ToHandlerFunc(v1Ctrl.HandleUserAPIKeysList(), userMW...))
		r.Post("/users/self/api-keys", chain.ToHandlerFunc(v1Ctrl.HandleUserAPIKeyCreate(), userMW...))
		r.Delete("/users/self/api-keys/{id}", chain.ToHandlerFunc(v1Ctrl.HandleUserAPIKeyDelete(), userMW...))

		// User administration (site-scoped)
		r.Get("/users", chain.ToHandlerFunc(v1Ctrl.HandleAdminUsersGetAll(), with(siteMW, a.mwPermission(permissions.SectionUsers, permissions.ActionView))...))
		r.Post("/users", chain.ToHandlerFunc(v1Ctrl.HandleAdminUserCreate(), with(siteMW, a.mwPermission(permissions.SectionUsers, permissions.ActionCreate))...))
		r.Put("/users/{id}", chain.ToHandlerFunc(v1Ctrl.HandleAdminUserUpdate(), with(siteMW, a.mwPermission(permissions.SectionUsers, permissions.ActionEdit))...))
		r.Delete("/users/{id}", chain.ToHandlerFunc(v1Ctrl.HandleAdminUserDelete(), with(siteMW, a.mwPermission(permissions.SectionUsers, permissions.ActionDelete))...))

		// Role ("Group" in the UI) administration (site-scoped)
		r.Get("/roles", chain.ToHandlerFunc(v1Ctrl.HandleRolesGetAll(), with(siteMW, a.mwPermission(permissions.SectionRoles, permissions.ActionView))...))
		r.Post("/roles", chain.ToHandlerFunc(v1Ctrl.HandleRoleCreate(), with(siteMW, a.mwPermission(permissions.SectionRoles, permissions.ActionCreate))...))
		r.Get("/roles/{id}", chain.ToHandlerFunc(v1Ctrl.HandleRoleGet(), with(siteMW, a.mwPermission(permissions.SectionRoles, permissions.ActionView))...))
		r.Put("/roles/{id}", chain.ToHandlerFunc(v1Ctrl.HandleRoleUpdate(), with(siteMW, a.mwPermission(permissions.SectionRoles, permissions.ActionEdit))...))
		r.Delete("/roles/{id}", chain.ToHandlerFunc(v1Ctrl.HandleRoleDelete(), with(siteMW, a.mwPermission(permissions.SectionRoles, permissions.ActionDelete))...))

		// Site settings administration (site-scoped)
		r.Get("/admin/settings", chain.ToHandlerFunc(v1Ctrl.HandleAdminSettingsGet(), with(siteMW, a.mwPermission(permissions.SectionSiteSettings, permissions.ActionView))...))
		r.Put("/admin/settings/{section}", chain.ToHandlerFunc(v1Ctrl.HandleAdminSettingsUpdate(), with(siteMW, a.mwPermission(permissions.SectionSiteSettings, permissions.ActionEdit))...))
		r.Delete("/admin/settings/{section}", chain.ToHandlerFunc(v1Ctrl.HandleAdminSettingsReset(), with(siteMW, a.mwPermission(permissions.SectionSiteSettings, permissions.ActionEdit))...))
		r.Post("/admin/settings/algolia/reindex", chain.ToHandlerFunc(v1Ctrl.HandleAdminSettingsAlgoliaReindex(), with(siteMW, a.mwPermission(permissions.SectionSiteSettings, permissions.ActionEdit))...))

		// Theming administration (site-scoped). The active theme's branding
		// assets are additionally served unauthenticated further below for
		// the login page.
		r.Get("/themes", chain.ToHandlerFunc(v1Ctrl.HandleThemesGetAll(), with(siteMW, a.mwPermission(permissions.SectionTheming, permissions.ActionView))...))
		r.Post("/themes", chain.ToHandlerFunc(v1Ctrl.HandleThemeCreate(), with(siteMW, a.mwPermission(permissions.SectionTheming, permissions.ActionCreate))...))
		r.Get("/themes/{id}", chain.ToHandlerFunc(v1Ctrl.HandleThemeGet(), with(siteMW, a.mwPermission(permissions.SectionTheming, permissions.ActionView))...))
		r.Put("/themes/{id}", chain.ToHandlerFunc(v1Ctrl.HandleThemeUpdate(), with(siteMW, a.mwPermission(permissions.SectionTheming, permissions.ActionEdit))...))
		r.Delete("/themes/{id}", chain.ToHandlerFunc(v1Ctrl.HandleThemeDelete(), with(siteMW, a.mwPermission(permissions.SectionTheming, permissions.ActionDelete))...))
		r.Post("/themes/{id}/assets/{kind}", chain.ToHandlerFunc(v1Ctrl.HandleThemeAssetUpload(), with(siteMW, a.mwPermission(permissions.SectionTheming, permissions.ActionEdit))...))
		r.Delete("/themes/{id}/assets/{kind}", chain.ToHandlerFunc(v1Ctrl.HandleThemeAssetDelete(), with(siteMW, a.mwPermission(permissions.SectionTheming, permissions.ActionEdit))...))
		r.Get("/themes/{id}/assets/{kind}", chain.ToHandlerFunc(v1Ctrl.HandleThemeAssetGet(), with(siteMW, a.mwPermission(permissions.SectionTheming, permissions.ActionView))...))
		r.Get("/theming/active", chain.ToHandlerFunc(v1Ctrl.HandleThemingActive(), with(siteMW, a.mwPermission(permissions.SectionTheming, permissions.ActionView))...))
		r.Put("/theming/active", chain.ToHandlerFunc(v1Ctrl.HandleThemingActiveSet(), with(siteMW, a.mwPermission(permissions.SectionTheming, permissions.ActionEdit))...))

		// Unauthenticated: branding images of the *active* theme only (the
		// route takes no theme ID, so non-active themes are not enumerable).
		// The login page renders these pre-auth.
		r.Get("/theming/assets/{kind}", chain.ToHandlerFunc(v1Ctrl.HandleThemingActiveAssetGet()))

		// Collection endpoints
		r.Get("/groups/all", chain.ToHandlerFunc(v1Ctrl.HandleGroupsGetAll(), userMW...))
		r.Post("/groups", chain.ToHandlerFunc(v1Ctrl.HandleGroupCreate(), with(siteMW, a.mwPermission(permissions.SectionCollections, permissions.ActionCreate))...))
		r.Get("/groups", chain.ToHandlerFunc(v1Ctrl.HandleGroupGet(), userMW...))
		r.Put("/groups", chain.ToHandlerFunc(v1Ctrl.HandleGroupUpdate(), with(userMW, a.mwPermission(permissions.SectionCollectionSettings, permissions.ActionEdit))...))
		r.Delete("/groups", chain.ToHandlerFunc(v1Ctrl.HandleGroupDelete(), with(userMW, a.mwPermission(permissions.SectionCollectionSettings, permissions.ActionDelete))...))

		// Collection export/import (Tools tab)
		r.Post("/group/exports", chain.ToHandlerFunc(v1Ctrl.HandleExportsCreate(), with(userMW, a.mwPermission(permissions.SectionTools, permissions.ActionCreate))...))
		r.Get("/group/exports", chain.ToHandlerFunc(v1Ctrl.HandleExportsList(), with(userMW, a.mwPermission(permissions.SectionTools, permissions.ActionView))...))
		r.Get("/group/exports/{id}", chain.ToHandlerFunc(v1Ctrl.HandleExportGet(), with(userMW, a.mwPermission(permissions.SectionTools, permissions.ActionView))...))
		r.Get("/group/exports/{id}/download", chain.ToHandlerFunc(v1Ctrl.HandleExportDownload(), with(userMW, a.mwPermission(permissions.SectionTools, permissions.ActionView))...))
		r.Delete("/group/exports/{id}", chain.ToHandlerFunc(v1Ctrl.HandleExportDelete(), with(userMW, a.mwPermission(permissions.SectionTools, permissions.ActionDelete))...))
		r.Post("/group/import", chain.ToHandlerFunc(v1Ctrl.HandleCollectionImport(), with(userMW, a.mwPermission(permissions.SectionTools, permissions.ActionCreate))...))

		// Statistics (Home dashboard)
		r.Get("/groups/statistics", chain.ToHandlerFunc(v1Ctrl.HandleGroupStatistics(), with(userMW, a.mwPermission(permissions.SectionStatistics, permissions.ActionView))...))
		r.Get("/groups/statistics/purchase-price", chain.ToHandlerFunc(v1Ctrl.HandleGroupStatisticsPriceOverTime(), with(userMW, a.mwPermission(permissions.SectionStatistics, permissions.ActionView))...))
		r.Get("/groups/statistics/locations", chain.ToHandlerFunc(v1Ctrl.HandleGroupStatisticsLocations(), with(userMW, a.mwPermission(permissions.SectionStatistics, permissions.ActionView))...))
		r.Get("/groups/statistics/tags", chain.ToHandlerFunc(v1Ctrl.HandleGroupStatisticsTags(), with(userMW, a.mwPermission(permissions.SectionStatistics, permissions.ActionView))...))

		// Action endpoints (Tools tab; they bulk-modify items, so both apply)
		r.Post("/actions/ensure-asset-ids", chain.ToHandlerFunc(v1Ctrl.HandleEnsureAssetID(), with(userMW, a.mwPermission(permissions.SectionTools, permissions.ActionEdit), a.mwPermission(permissions.SectionItems, permissions.ActionEdit))...))
		r.Post("/actions/ensure-import-refs", chain.ToHandlerFunc(v1Ctrl.HandleEnsureImportRefs(), with(userMW, a.mwPermission(permissions.SectionTools, permissions.ActionEdit), a.mwPermission(permissions.SectionItems, permissions.ActionEdit))...))

		// Tags endpoints
		r.Get("/tags", chain.ToHandlerFunc(v1Ctrl.HandleTagsGetAll(), with(userMW, a.mwPermission(permissions.SectionTags, permissions.ActionView))...))
		r.Post("/tags", chain.ToHandlerFunc(v1Ctrl.HandleTagsCreate(), with(userMW, a.mwPermission(permissions.SectionTags, permissions.ActionCreate))...))
		r.Get("/tags/{id}", chain.ToHandlerFunc(v1Ctrl.HandleTagGet(), with(userMW, a.mwPermission(permissions.SectionTags, permissions.ActionView))...))
		r.Put("/tags/{id}", chain.ToHandlerFunc(v1Ctrl.HandleTagUpdate(), with(userMW, a.mwPermission(permissions.SectionTags, permissions.ActionEdit))...))
		r.Delete("/tags/{id}", chain.ToHandlerFunc(v1Ctrl.HandleTagDelete(), with(userMW, a.mwPermission(permissions.SectionTags, permissions.ActionDelete))...))

		// Entity Type endpoints
		r.Get("/entity-types", chain.ToHandlerFunc(v1Ctrl.HandleEntityTypeGetAll(), with(userMW, a.mwPermission(permissions.SectionEntityTypes, permissions.ActionView))...))
		r.Post("/entity-types", chain.ToHandlerFunc(v1Ctrl.HandleEntityTypeCreate(), with(userMW, a.mwPermission(permissions.SectionEntityTypes, permissions.ActionCreate))...))
		r.Put("/entity-types/{id}", chain.ToHandlerFunc(v1Ctrl.HandleEntityTypeUpdate(), with(userMW, a.mwPermission(permissions.SectionEntityTypes, permissions.ActionEdit))...))
		r.Delete("/entity-types/{id}", chain.ToHandlerFunc(v1Ctrl.HandleEntityTypeDelete(), with(userMW, a.mwPermission(permissions.SectionEntityTypes, permissions.ActionDelete))...))

		// Entity endpoints (primary). Routes shared by items and locations use
		// mwPermissionAny for fail-fast; the handler resolves the entity's real
		// section (items vs locations) and enforces precisely.
		r.Get("/entities", chain.ToHandlerFunc(v1Ctrl.HandleEntitiesGetAll(), with(userMW, a.mwPermissionAny(entitySections, permissions.ActionView))...))
		r.Post("/entities", chain.ToHandlerFunc(v1Ctrl.HandleEntitiesCreate(), with(userMW, a.mwPermissionAny(entitySections, permissions.ActionCreate))...))
		r.Post("/entities/import", chain.ToHandlerFunc(v1Ctrl.HandleEntitiesImport(), with(userMW, a.mwPermission(permissions.SectionTools, permissions.ActionCreate))...))
		r.Get("/entities/export", chain.ToHandlerFunc(v1Ctrl.HandleEntitiesExport(), with(userMW, a.mwPermission(permissions.SectionTools, permissions.ActionView))...))
		r.Get("/entities/fields", chain.ToHandlerFunc(v1Ctrl.HandleGetAllCustomFieldNames(), with(userMW, a.mwPermissionAny(entitySections, permissions.ActionView))...))
		r.Get("/entities/fields/values", chain.ToHandlerFunc(v1Ctrl.HandleGetAllCustomFieldValues(), with(userMW, a.mwPermissionAny(entitySections, permissions.ActionView))...))
		r.Get("/entities/tree", chain.ToHandlerFunc(v1Ctrl.HandleLocationTreeQuery(), with(userMW, a.mwPermission(permissions.SectionLocations, permissions.ActionView))...))

		r.Get("/entities/{id}", chain.ToHandlerFunc(v1Ctrl.HandleEntityGet(), with(userMW, a.mwPermissionAny(entitySections, permissions.ActionView))...))
		r.Get("/entities/{id}/path", chain.ToHandlerFunc(v1Ctrl.HandleEntityFullPath(), with(userMW, a.mwPermissionAny(entitySections, permissions.ActionView))...))
		r.Put("/entities/{id}", chain.ToHandlerFunc(v1Ctrl.HandleEntityUpdate(), with(userMW, a.mwPermissionAny(entitySections, permissions.ActionEdit))...))
		r.Patch("/entities/{id}", chain.ToHandlerFunc(v1Ctrl.HandleEntityPatch(), with(userMW, a.mwPermissionAny(entitySections, permissions.ActionEdit))...))
		r.Delete("/entities/{id}", chain.ToHandlerFunc(v1Ctrl.HandleEntityDelete(), with(userMW, a.mwPermissionAny(entitySections, permissions.ActionDelete))...))
		r.Post("/entities/{id}/duplicate", chain.ToHandlerFunc(v1Ctrl.HandleEntityDuplicate(), with(userMW, a.mwPermissionAny(entitySections, permissions.ActionCreate))...))

		// Entity attachment endpoints (attachments are part of the parent
		// entity's surface: writes ride on Edit)
		r.Post("/entities/{id}/attachments", chain.ToHandlerFunc(v1Ctrl.HandleEntityAttachmentCreate(), with(userMW, a.mwPermissionAny(entitySections, permissions.ActionEdit))...))
		r.Post("/entities/{id}/attachments/external", chain.ToHandlerFunc(v1Ctrl.HandleEntityAttachmentExternalCreate(), with(userMW, a.mwPermissionAny(entitySections, permissions.ActionEdit))...))
		r.Put("/entities/{id}/attachments/{attachment_id}", chain.ToHandlerFunc(v1Ctrl.HandleEntityAttachmentUpdate(), with(userMW, a.mwPermissionAny(entitySections, permissions.ActionEdit))...))
		r.Delete("/entities/{id}/attachments/{attachment_id}", chain.ToHandlerFunc(v1Ctrl.HandleEntityAttachmentDelete(), with(userMW, a.mwPermissionAny(entitySections, permissions.ActionEdit))...))

		// Entity maintenance endpoints
		r.Get("/entities/{id}/maintenance", chain.ToHandlerFunc(v1Ctrl.HandleMaintenanceLogGet(), with(userMW, a.mwPermission(permissions.SectionMaintenance, permissions.ActionView))...))
		r.Post("/entities/{id}/maintenance", chain.ToHandlerFunc(v1Ctrl.HandleMaintenanceEntryCreate(), with(userMW, a.mwPermission(permissions.SectionMaintenance, permissions.ActionCreate))...))

		r.Get("/assets/{id}", chain.ToHandlerFunc(v1Ctrl.HandleAssetGet(), with(userMW, a.mwPermission(permissions.SectionItems, permissions.ActionView))...))

		// Entity Templates
		r.Get("/templates", chain.ToHandlerFunc(v1Ctrl.HandleEntityTemplatesGetAll(), with(userMW, a.mwPermission(permissions.SectionTemplates, permissions.ActionView))...))
		r.Post("/templates", chain.ToHandlerFunc(v1Ctrl.HandleEntityTemplatesCreate(), with(userMW, a.mwPermission(permissions.SectionTemplates, permissions.ActionCreate))...))
		r.Get("/templates/{id}", chain.ToHandlerFunc(v1Ctrl.HandleEntityTemplatesGet(), with(userMW, a.mwPermission(permissions.SectionTemplates, permissions.ActionView))...))
		r.Put("/templates/{id}", chain.ToHandlerFunc(v1Ctrl.HandleEntityTemplatesUpdate(), with(userMW, a.mwPermission(permissions.SectionTemplates, permissions.ActionEdit))...))
		r.Delete("/templates/{id}", chain.ToHandlerFunc(v1Ctrl.HandleEntityTemplatesDelete(), with(userMW, a.mwPermission(permissions.SectionTemplates, permissions.ActionDelete))...))
		r.Post("/templates/{id}/create-item", chain.ToHandlerFunc(v1Ctrl.HandleEntityTemplatesCreateItem(), with(userMW, a.mwPermission(permissions.SectionTemplates, permissions.ActionView), a.mwPermissionAny(entitySections, permissions.ActionCreate))...))

		// Maintenance
		r.Get("/maintenance", chain.ToHandlerFunc(v1Ctrl.HandleMaintenanceGetAll(), with(userMW, a.mwPermission(permissions.SectionMaintenance, permissions.ActionView))...))
		r.Put("/maintenance/{id}", chain.ToHandlerFunc(v1Ctrl.HandleMaintenanceEntryUpdate(), with(userMW, a.mwPermission(permissions.SectionMaintenance, permissions.ActionEdit))...))
		r.Delete("/maintenance/{id}", chain.ToHandlerFunc(v1Ctrl.HandleMaintenanceEntryDelete(), with(userMW, a.mwPermission(permissions.SectionMaintenance, permissions.ActionDelete))...))

		// Notifiers
		r.Get("/notifiers", chain.ToHandlerFunc(v1Ctrl.HandleGetUserNotifiers(), with(userMW, a.mwPermission(permissions.SectionNotifiers, permissions.ActionView))...))
		r.Post("/notifiers", chain.ToHandlerFunc(v1Ctrl.HandleCreateNotifier(), with(userMW, a.mwPermission(permissions.SectionNotifiers, permissions.ActionCreate))...))
		r.Put("/notifiers/{id}", chain.ToHandlerFunc(v1Ctrl.HandleUpdateNotifier(), with(userMW, a.mwPermission(permissions.SectionNotifiers, permissions.ActionEdit))...))
		r.Delete("/notifiers/{id}", chain.ToHandlerFunc(v1Ctrl.HandleDeleteNotifier(), with(userMW, a.mwPermission(permissions.SectionNotifiers, permissions.ActionDelete))...))
		r.Post("/notifiers/test", chain.ToHandlerFunc(v1Ctrl.HandlerNotifierTest(), with(userMW, a.mwPermission(permissions.SectionNotifiers, permissions.ActionCreate), a.notifierTestLimiter.middleware)...))

		// Asset-Like endpoints
		assetMW := []errchain.Middleware{
			a.mwAuthToken,
			a.mwTenant,
			a.mwRoles(RoleModeOr, authroles.RoleUser.String(), authroles.RoleAttachments.String()),
		}

		r.Get("/products/search-from-barcode", chain.ToHandlerFunc(v1Ctrl.HandleProductSearchFromBarcode(), with(userMW, a.mwPermissionAny(entitySections, permissions.ActionCreate))...))

		// Unauthenticated signed-URL thumbnails for external search consumers.
		// The handler 404s unless public image URLs are enabled and the HMAC
		// signature matches.
		r.Get("/public/attachments/{attachment_id}", chain.ToHandlerFunc(v1Ctrl.HandlePublicAttachmentGet()))

		r.Get("/qrcode", chain.ToHandlerFunc(v1Ctrl.HandleGenerateQRCode(), with(assetMW, a.mwPermissionAny(entitySections, permissions.ActionView))...))
		r.Get(
			"/entities/{id}/attachments/{attachment_id}",
			chain.ToHandlerFunc(v1Ctrl.HandleEntityAttachmentGet(), with(assetMW, a.mwPermissionAny(entitySections, permissions.ActionView))...),
		)

		// Labelmaker
		r.Get("/labelmaker/entity/{id}", chain.ToHandlerFunc(v1Ctrl.HandleGetItemLabel(), with(userMW, a.mwPermissionAny(entitySections, permissions.ActionView))...))
		r.Get("/labelmaker/location/{id}", chain.ToHandlerFunc(v1Ctrl.HandleGetLocationLabel(), with(userMW, a.mwPermission(permissions.SectionLocations, permissions.ActionView))...))
		r.Get("/labelmaker/item/{id}", chain.ToHandlerFunc(v1Ctrl.HandleGetItemLabel(), with(userMW, a.mwPermission(permissions.SectionItems, permissions.ActionView))...))
		r.Get("/labelmaker/asset/{id}", chain.ToHandlerFunc(v1Ctrl.HandleGetAssetLabel(), with(userMW, a.mwPermission(permissions.SectionItems, permissions.ActionView))...))

		// Reporting Services (Tools tab)
		r.Get("/reporting/bill-of-materials", chain.ToHandlerFunc(v1Ctrl.HandleBillOfMaterialsExport(), with(userMW, a.mwPermission(permissions.SectionTools, permissions.ActionView))...))

		// OpenTelemetry proxy endpoint for frontend telemetry (requires auth)
		if a.otel != nil && a.otel.IsEnabled() && a.conf.Otel.ProxyEnabled {
			r.Post("/telemetry", chain.ToHandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
				a.otel.ProxyHandler().ServeHTTP(w, r)
				return nil
			}, userMW...))
		}

		r.NotFound(http.NotFound)
	})
}
