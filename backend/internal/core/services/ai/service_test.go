package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	_ "github.com/sysadminsmedia/homebox/backend/pkgs/cgofreesqlite"
)

func newTestService(t *testing.T, llmURL string) (*Service, *repo.AllRepos, repo.Group) {
	t.Helper()

	client, err := ent.Open("sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared&_fk=1&_time_format=sqlite")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	bus := eventbus.New()
	repos := repo.New(client, bus, config.Storage{PrefixPath: "/", ConnString: "mem://"}, "mem://{{ .Topic }}",
		repo.StaticThumbnail(config.Thumbnail{}))

	group, err := repos.Groups.GroupCreate(context.Background(), "ai-test-group")
	if err != nil {
		t.Fatalf("creating group: %v", err)
	}

	svc := NewService(repos, func() config.AIConf {
		return config.AIConf{Enabled: true, BaseURL: llmURL, Model: "test-model"}
	})
	return svc, repos, group
}

// testPNG renders a small valid image for prepareImage to chew on.
func testPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatalf("encoding test png: %v", err)
	}
	return buf.Bytes()
}

func llmStub(t *testing.T, content string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := json.Marshal(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"content": content}}},
		})
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAnalyzeImagesParsesAndFlagsDuplicates(t *testing.T) {
	srv := llmStub(t, `{"items":[
		{"name":"DeWalt Drill","quantity":1,"description":"Cordless drill","manufacturer":"DeWalt","modelNumber":"DCD771","serialNumber":"SN-123","purchasePrice":0,"purchaseFrom":"","notes":""},
		{"name":"Hammer","quantity":2,"description":"Claw hammer","manufacturer":"","modelNumber":"","serialNumber":"","purchasePrice":0,"purchaseFrom":"","notes":""}
	]}`)

	svc, repos, group := newTestService(t, srv.URL)
	ctx := context.Background()

	// Seed an existing item with the same serial (different case).
	existing, err := repos.Entities.Create(ctx, group.ID, repo.EntityCreate{
		Name:         "Old Drill",
		SerialNumber: "sn-123",
	})
	if err != nil {
		t.Fatalf("seeding entity: %v", err)
	}

	items, err := svc.AnalyzeImages(ctx, group.ID, [][]byte{testPNG(t)}, AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeImages: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("items: got %d, want 2", len(items))
	}
	if items[0].Name != "DeWalt Drill" || items[0].Quantity != 1 {
		t.Errorf("first item: got %+v", items[0].DetectedItem)
	}
	if items[0].Duplicate == nil {
		t.Fatal("first item: expected duplicate match by serial")
	}
	if items[0].Duplicate.ID != existing.ID || items[0].Duplicate.Name != "Old Drill" {
		t.Errorf("duplicate: got %+v", items[0].Duplicate)
	}
	if items[1].Duplicate != nil {
		t.Errorf("second item (no serial): unexpected duplicate %+v", items[1].Duplicate)
	}
}

func TestAnalyzeImagesDisabled(t *testing.T) {
	svc, _, group := newTestService(t, "")
	svc.conf = func() config.AIConf { return config.AIConf{} }

	if _, err := svc.AnalyzeImages(context.Background(), group.ID, [][]byte{testPNG(t)}, AnalyzeOptions{}); err != ErrDisabled {
		t.Errorf("got %v, want ErrDisabled", err)
	}
}

func TestFilterSuggestions(t *testing.T) {
	current := map[string]string{
		"name":          "Old Name",
		"description":   "",
		"manufacturer":  "DeWalt",
		"modelNumber":   "",
		"serialNumber":  "",
		"purchasePrice": "",
		"purchaseFrom":  "",
		"notes":         "",
	}
	suggested := map[string]string{
		"name":          "New Name",
		"description":   "A cordless drill",
		"manufacturer":  "DeWalt",
		"modelNumber":   "unknown",
		"serialNumber":  "SN-9",
		"purchasePrice": "not-a-number",
		"purchaseFrom":  "Home Depot",
		"notes":         "",
	}

	got := filterSuggestions(current, suggested, false)

	want := map[string]string{
		"description":  "A cordless drill",
		"serialNumber": "SN-9",
		"purchaseFrom": "Home Depot",
	}
	if len(got) != len(want) {
		t.Fatalf("suggestions: got %+v, want fields %v", got, want)
	}
	for _, s := range got {
		if want[s.Field] != s.Suggested {
			t.Errorf("field %s: got %q, want %q", s.Field, s.Suggested, want[s.Field])
		}
	}

	// With overwrite, filled fields become suggestible; equal values and
	// junk values stay excluded.
	got = filterSuggestions(current, suggested, true)
	fields := map[string]bool{}
	for _, s := range got {
		fields[s.Field] = true
	}
	if !fields["name"] {
		t.Error("overwrite: expected name suggestion")
	}
	if fields["manufacturer"] {
		t.Error("overwrite: manufacturer equals current and must be excluded")
	}
	if fields["modelNumber"] {
		t.Error("'unknown' must be treated as empty")
	}
	if fields["purchasePrice"] {
		t.Error("unparseable price must be excluded")
	}
}
