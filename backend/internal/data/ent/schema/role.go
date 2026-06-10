package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/schema/mixins"
)

// Role is a named bundle of permissions (labeled "Group" in the UI). Users may
// hold multiple roles; their effective permissions are the union of all grants.
type Role struct {
	ent.Schema
}

func (Role) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.BaseMixin{},
	}
}

func (Role) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(255).
			NotEmpty().
			Unique(),
		field.String("description").
			MaxLen(1000).
			Optional(),
		// is_super_admin short-circuits permission evaluation; the seeded
		// Super Admin role cannot be edited or deleted.
		field.Bool("is_super_admin").
			Default(false).
			Immutable(),
	}
}

func (Role) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("permissions", RolePermission.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.From("users", User.Type).
			Ref("roles"),
	}
}
