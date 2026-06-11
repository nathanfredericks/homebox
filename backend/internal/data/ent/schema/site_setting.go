package schema

import (
	"encoding/json"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/schema/mixins"
)

// SiteSetting holds one site-wide settings section as a sparse JSON override
// document. The key is the section name (e.g. "thumbnail", "algolia"); the
// value contains only the fields an administrator has explicitly saved, so
// anything absent falls back to the environment/default configuration.
type SiteSetting struct {
	ent.Schema
}

func (SiteSetting) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.BaseMixin{},
	}
}

// Fields of the SiteSetting.
func (SiteSetting) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").
			NotEmpty().
			Unique(),
		field.JSON("value", json.RawMessage{}),
	}
}
