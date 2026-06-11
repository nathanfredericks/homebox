package repo

import (
	"context"
	"encoding/json"

	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/sitesetting"
)

// SiteSettingsRepository persists site-wide settings sections. Each row holds
// one section keyed by name (e.g. "thumbnail") whose value is a sparse JSON
// override document; absent fields fall back to environment configuration.
type SiteSettingsRepository struct {
	db *ent.Client
}

func NewSiteSettingsRepository(db *ent.Client) *SiteSettingsRepository {
	return &SiteSettingsRepository{db: db}
}

// GetAll returns every stored section keyed by section name.
func (r *SiteSettingsRepository) GetAll(ctx context.Context) (map[string]json.RawMessage, error) {
	rows, err := r.db.SiteSetting.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	out := make(map[string]json.RawMessage, len(rows))
	for _, row := range rows {
		out[row.Key] = row.Value
	}
	return out, nil
}

// Upsert stores the override document for one section, replacing any existing row.
func (r *SiteSettingsRepository) Upsert(ctx context.Context, key string, value json.RawMessage) error {
	n, err := r.db.SiteSetting.Update().
		Where(sitesetting.Key(key)).
		SetValue(value).
		Save(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	return r.db.SiteSetting.Create().
		SetKey(key).
		SetValue(value).
		Exec(ctx)
}

// Delete removes a section's override, restoring environment/default values.
// Deleting a section that has no override is a no-op.
func (r *SiteSettingsRepository) Delete(ctx context.Context, key string) error {
	_, err := r.db.SiteSetting.Delete().
		Where(sitesetting.Key(key)).
		Exec(ctx)
	return err
}
