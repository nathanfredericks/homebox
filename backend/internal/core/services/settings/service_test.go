package settings

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	_ "github.com/sysadminsmedia/homebox/backend/pkgs/cgofreesqlite"
)

func newTestService(t *testing.T, cfg *config.Config) *Service {
	t.Helper()

	client, err := ent.Open("sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared&_fk=1&_time_format=sqlite")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	svc, err := New(context.Background(), repo.NewSiteSettingsRepository(client), cfg, nil)
	if err != nil {
		t.Fatalf("creating service: %v", err)
	}
	return svc
}

// baseConfig simulates an environment-parsed config. The web-managed sections
// are expected to be IGNORED by the service (its base comes from compiled
// defaults); only the env-only Options fields should carry over.
func baseConfig() *config.Config {
	return &config.Config{
		Thumbnail: config.Thumbnail{Enabled: false, Width: 300, Height: 300},
		Options:   config.Options{TrustProxy: false, AllowAnalytics: true, CurrencyConfig: "/tmp/currencies.json"},
		Algolia:   config.AlgoliaConf{IndexName: "env-index", AdminAPIKey: "env-secret"},
	}
}

func TestEnvironmentIsIgnoredForWebSections(t *testing.T) {
	svc := newTestService(t, baseConfig())

	// The env config set thumbnail to 300x300/disabled and a custom Algolia
	// index, but the base must be the compiled defaults.
	got := svc.Get()
	if got.Thumbnail.Width != 500 || got.Thumbnail.Height != 500 || !got.Thumbnail.Enabled {
		t.Errorf("thumbnail: got %+v, want compiled defaults (enabled, 500x500)", got.Thumbnail)
	}
	if got.Algolia.IndexName != "homebox-items" {
		t.Errorf("algolia index: got %q, want default homebox-items", got.Algolia.IndexName)
	}
	if got.Algolia.AdminAPIKey != "" {
		t.Errorf("algolia secret: got %q, want empty (env value ignored)", got.Algolia.AdminAPIKey)
	}
}

func TestEnvOnlyFieldsComeFromEnvironment(t *testing.T) {
	svc := newTestService(t, baseConfig())

	got := svc.Get().Options
	if !got.AllowAnalytics {
		t.Error("allowAnalytics is env-only and must carry the environment value")
	}
	if got.CurrencyConfig != "/tmp/currencies.json" {
		t.Errorf("currencyConfig: got %q, want environment value", got.CurrencyConfig)
	}
}

func TestSparseOverrideLayering(t *testing.T) {
	svc := newTestService(t, baseConfig())
	ctx := context.Background()

	// Override only the width; height keeps its default value.
	if err := svc.UpdateSection(ctx, SectionThumbnail, json.RawMessage(`{"width":700}`)); err != nil {
		t.Fatalf("update: %v", err)
	}

	got := svc.Get().Thumbnail
	if got.Width != 700 {
		t.Errorf("width: got %d, want 700 (database override)", got.Width)
	}
	if got.Height != 500 {
		t.Errorf("height: got %d, want 500 (default fallback)", got.Height)
	}
	if !got.Enabled {
		t.Error("enabled: lost default value")
	}
}

func TestResetSectionRestoresDefaults(t *testing.T) {
	svc := newTestService(t, baseConfig())
	ctx := context.Background()

	if err := svc.UpdateSection(ctx, SectionThumbnail, json.RawMessage(`{"width":700}`)); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := svc.ResetSection(ctx, SectionThumbnail); err != nil {
		t.Fatalf("reset: %v", err)
	}

	if got := svc.Get().Thumbnail.Width; got != 500 {
		t.Errorf("width after reset: got %d, want 500 (compiled default)", got)
	}
}

func TestSecretSentinelKeepsCurrentValue(t *testing.T) {
	svc := newTestService(t, baseConfig())
	ctx := context.Background()

	// Save a new secret, then echo the sentinel back: the saved secret stays.
	if err := svc.UpdateSection(ctx, SectionAlgolia, json.RawMessage(`{"adminApiKey":"db-secret","enabled":true}`)); err != nil {
		t.Fatalf("update: %v", err)
	}
	if got := svc.Get().Algolia.AdminAPIKey; got != "db-secret" {
		t.Fatalf("adminApiKey: got %q, want db-secret", got)
	}

	if err := svc.UpdateSection(ctx, SectionAlgolia, json.RawMessage(`{"adminApiKey":"[REDACTED]","enabled":false}`)); err != nil {
		t.Fatalf("update with sentinel: %v", err)
	}
	got := svc.Get().Algolia
	if got.AdminAPIKey != "db-secret" {
		t.Errorf("adminApiKey: got %q, want preserved db-secret", got.AdminAPIKey)
	}
	if got.Enabled {
		t.Error("enabled: non-secret field should have been updated alongside")
	}
}

func TestRedactionOnMarshal(t *testing.T) {
	svc := newTestService(t, baseConfig())
	ctx := context.Background()

	if err := svc.UpdateSection(ctx, SectionAlgolia, json.RawMessage(`{"adminApiKey":"db-secret"}`)); err != nil {
		t.Fatalf("update: %v", err)
	}

	out, err := json.Marshal(svc.Get())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var doc map[string]map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := doc["algolia"]["adminApiKey"]; got != RedactedSentinel {
		t.Errorf("marshaled adminApiKey: got %v, want sentinel", got)
	}
}

func TestRejectsUnknownSectionAndFields(t *testing.T) {
	svc := newTestService(t, baseConfig())
	ctx := context.Background()

	if err := svc.UpdateSection(ctx, "nope", json.RawMessage(`{}`)); !errors.Is(err, ErrUnknownSection) {
		t.Errorf("unknown section: got %v, want ErrUnknownSection", err)
	}
	if err := svc.ResetSection(ctx, "nope"); !errors.Is(err, ErrUnknownSection) {
		t.Errorf("unknown section reset: got %v, want ErrUnknownSection", err)
	}
	if err := svc.UpdateSection(ctx, SectionThumbnail, json.RawMessage(`{"bogus":1}`)); !errors.Is(err, ErrInvalidPayload) {
		t.Errorf("unknown field: got %v, want ErrInvalidPayload", err)
	}
	if err := svc.UpdateSection(ctx, SectionThumbnail, json.RawMessage(`{"width":"wide"}`)); !errors.Is(err, ErrInvalidPayload) {
		t.Errorf("type mismatch: got %v, want ErrInvalidPayload", err)
	}
	if err := svc.UpdateSection(ctx, SectionThumbnail, json.RawMessage(`not json`)); !errors.Is(err, ErrInvalidPayload) {
		t.Errorf("malformed json: got %v, want ErrInvalidPayload", err)
	}
}

func TestEnvOnlyFieldsAreStripped(t *testing.T) {
	svc := newTestService(t, baseConfig())
	ctx := context.Background()

	if err := svc.UpdateSection(ctx, SectionOptions, json.RawMessage(`{"trustProxy":true,"autoIncrementAssetId":false}`)); err != nil {
		t.Fatalf("update: %v", err)
	}
	got := svc.Get().Options
	if got.TrustProxy {
		t.Error("trustProxy is env-only and must not be overridable from the database")
	}
	if got.AutoIncrementAssetID {
		t.Error("autoIncrementAssetId should have been overridden to false")
	}
}

func TestOnChangeListenerFires(t *testing.T) {
	svc := newTestService(t, baseConfig())
	ctx := context.Background()

	var oldW, newW int
	svc.OnChange(func(old, new Resolved) {
		oldW, newW = old.Thumbnail.Width, new.Thumbnail.Width
	})

	if err := svc.UpdateSection(ctx, SectionThumbnail, json.RawMessage(`{"width":900}`)); err != nil {
		t.Fatalf("update: %v", err)
	}
	if oldW != 500 || newW != 900 {
		t.Errorf("listener saw old=%d new=%d, want 500/900", oldW, newW)
	}
}

func TestOverridesSurviveServiceRestart(t *testing.T) {
	cfg := baseConfig()

	client, err := ent.Open("sqlite3", "file:restart-test?mode=memory&cache=shared&_fk=1&_time_format=sqlite")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	r := repo.NewSiteSettingsRepository(client)
	ctx := context.Background()

	svc, err := New(ctx, r, cfg, nil)
	if err != nil {
		t.Fatalf("creating service: %v", err)
	}
	if err := svc.UpdateSection(ctx, SectionThumbnail, json.RawMessage(`{"width":700}`)); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Simulate a restart: a fresh service over the same database.
	svc2, err := New(ctx, r, cfg, nil)
	if err != nil {
		t.Fatalf("recreating service: %v", err)
	}
	if got := svc2.Get().Thumbnail.Width; got != 700 {
		t.Errorf("width after restart: got %d, want 700 (database wins over defaults)", got)
	}
}

func TestAISecretSentinelRoundTrip(t *testing.T) {
	svc := newTestService(t, baseConfig())
	ctx := context.Background()

	if err := svc.UpdateSection(ctx, SectionAI, json.RawMessage(`{"apiKey":"sk-test","enabled":true,"model":"gpt-4o-mini"}`)); err != nil {
		t.Fatalf("update: %v", err)
	}
	if got := svc.Get().AI.APIKey; got != "sk-test" {
		t.Fatalf("apiKey: got %q, want sk-test", got)
	}

	// Echoing the sentinel back keeps the stored key while other fields update.
	if err := svc.UpdateSection(ctx, SectionAI, json.RawMessage(`{"apiKey":"[REDACTED]","model":"gpt-4o"}`)); err != nil {
		t.Fatalf("update with sentinel: %v", err)
	}
	got := svc.Get().AI
	if got.APIKey != "sk-test" {
		t.Errorf("apiKey: got %q, want preserved sk-test", got.APIKey)
	}
	if got.Model != "gpt-4o" {
		t.Errorf("model: got %q, want gpt-4o", got.Model)
	}

	out, err := json.Marshal(svc.Get())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var doc map[string]map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := doc["ai"]["apiKey"]; got != RedactedSentinel {
		t.Errorf("marshaled apiKey: got %v, want sentinel", got)
	}
}

func TestUpdateSectionMergesWithExistingOverride(t *testing.T) {
	svc := newTestService(t, baseConfig())
	ctx := context.Background()

	if err := svc.UpdateSection(ctx, SectionThumbnail, json.RawMessage(`{"width":700}`)); err != nil {
		t.Fatalf("first update: %v", err)
	}
	if err := svc.UpdateSection(ctx, SectionThumbnail, json.RawMessage(`{"height":900}`)); err != nil {
		t.Fatalf("second update: %v", err)
	}

	got := svc.Get().Thumbnail
	if got.Width != 700 {
		t.Errorf("width: got %d, want 700 (kept from first update)", got.Width)
	}
	if got.Height != 900 {
		t.Errorf("height: got %d, want 900 (from second update)", got.Height)
	}
}
