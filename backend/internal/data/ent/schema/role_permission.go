package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/schema/mixins"
)

// RolePermission is one grant row in a role's permission matrix: a section
// (validated in Go against permissions.AllSections), an optional collection
// scope (nil = all collections, or the site scope for site-level sections),
// and the four basic actions.
type RolePermission struct {
	ent.Schema
}

func (RolePermission) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.BaseMixin{},
	}
}

func (RolePermission) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("role_id", uuid.UUID{}),
		field.String("section").
			MaxLen(64).
			NotEmpty(),
		field.UUID("collection_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.Bool("can_view").Default(false),
		field.Bool("can_create").Default(false),
		field.Bool("can_edit").Default(false),
		field.Bool("can_delete").Default(false),
	}
}

func (RolePermission) Indexes() []ent.Index {
	// NULL collection_ids are distinct in unique indexes on both dialects;
	// role updates replace the full permission set in one transaction so
	// duplicate all-collections rows cannot accumulate.
	return []ent.Index{
		index.Fields("role_id", "section", "collection_id").Unique(),
	}
}

func (RolePermission) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("role", Role.Type).
			Ref("permissions").
			Field("role_id").
			Unique().
			Required(),
		edge.From("collection", Group.Type).
			Ref("role_permissions").
			Field("collection_id").
			Unique(),
	}
}
