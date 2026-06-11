package algolia

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/data/types"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	"github.com/sysadminsmedia/homebox/backend/pkgs/hasher"
)

func sampleEntity(parentID, thumbID uuid.UUID) repo.EntityOut {
	parent := repo.EntitySummary{ID: parentID, Name: "Shelf"}
	return repo.EntityOut{
		// EntityOut shadows AssetID and Parent from the embedded summary;
		// mapEntityOut fills the outer fields, so the builder reads those.
		AssetID: repo.AssetID(42),
		Parent:  &parent,
		EntitySummary: repo.EntitySummary{
			ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Name:        "Drill",
			Description: "Cordless drill",
			Quantity:    1,
			Insured:     true,
			Archived:    false,
			CreatedAt:   time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 1, 3, 3, 4, 5, 0, time.UTC),
			Tags:        []repo.TagSummary{},
			ThumbnailId: &thumbID,
		},
		SoldDate:     types.DateFromTime(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)),
		PurchaseDate: types.Date{},
		Manufacturer: "DeWalt",
	}
}

func TestBuildRecord_FullShape(t *testing.T) {
	hasher.SetAPIKeyPepper([]byte("test-pepper-test-pepper-test-pepper!"))

	gid := uuid.New()
	parentID := uuid.New()
	thumbID := uuid.New()
	paths := map[uuid.UUID]string{parentID: "Garage > Shelf"}

	rec := buildRecord(sampleEntity(parentID, thumbID), gid, paths, nil, "https://box.example.com")

	if rec["objectID"] != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("objectID: got %v", rec["objectID"])
	}
	if rec["groupId"] != gid.String() {
		t.Errorf("groupId: got %v", rec["groupId"])
	}
	if rec["assetId"] != "000-042" {
		t.Errorf("assetId: got %v, want formatted 000-042", rec["assetId"])
	}
	if rec["location"] != "Garage > Shelf" {
		t.Errorf("location: got %v, want flattened parent path", rec["location"])
	}
	if rec["soldTime"] != "2026-02-01" {
		t.Errorf("soldTime: got %v", rec["soldTime"])
	}
	if rec["purchaseTime"] != nil {
		t.Errorf("purchaseTime: got %v, want nil for zero date", rec["purchaseTime"])
	}
	want := "https://box.example.com/api/v1/public/attachments/" + thumbID.String() + "?sig=" + hasher.SignPublicAttachmentID(thumbID)
	if rec["thumbnailUrl"] != want {
		t.Errorf("thumbnailUrl: got %v, want %v", rec["thumbnailUrl"], want)
	}
}

func TestBuildRecord_AllowlistFiltersFields(t *testing.T) {
	gid := uuid.New()
	allow := parseFieldAllowlist("name, Description, bogusField")

	rec := buildRecord(sampleEntity(uuid.New(), uuid.New()), gid, nil, allow, "")

	if _, ok := rec["name"]; !ok {
		t.Error("name should be included")
	}
	if _, ok := rec["description"]; !ok {
		t.Error("description should be included (case-insensitive match)")
	}
	if _, ok := rec["manufacturer"]; ok {
		t.Error("manufacturer should be filtered out")
	}
	if rec["objectID"] == nil || rec["groupId"] == nil {
		t.Error("objectID and groupId are always included")
	}
}

func TestBuildRecord_NoPublicBaseOmitsThumbnail(t *testing.T) {
	rec := buildRecord(sampleEntity(uuid.New(), uuid.New()), uuid.New(), nil, nil, "")
	if rec["thumbnailUrl"] != nil {
		t.Errorf("thumbnailUrl: got %v, want nil when public images are off", rec["thumbnailUrl"])
	}
}

func TestPublicBaseURL(t *testing.T) {
	cases := []struct {
		name     string
		conf     config.AlgoliaConf
		hostname string
		want     string
	}{
		{"disabled", config.AlgoliaConf{PublicImageURLs: false, PublicBaseURL: "https://x.com"}, "h.com", ""},
		{"explicit base", config.AlgoliaConf{PublicImageURLs: true, PublicBaseURL: "https://x.com/"}, "h.com", "https://x.com"},
		{"hostname fallback", config.AlgoliaConf{PublicImageURLs: true}, "h.com", "https://h.com"},
		{"scheme kept", config.AlgoliaConf{PublicImageURLs: true, PublicBaseURL: "http://local:7745"}, "", "http://local:7745"},
		{"nothing configured", config.AlgoliaConf{PublicImageURLs: true}, "", ""},
	}
	for _, tc := range cases {
		if got := publicBaseURL(tc.conf, tc.hostname); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}
