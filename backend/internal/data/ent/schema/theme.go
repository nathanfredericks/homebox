package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/schema/mixins"
)

// Theme is an admin-created instance theme: a core color palette plus
// platform fonts and whitelabel branding. The site-wide active theme is
// referenced from the "theming" site settings row, which may also point at a
// built-in theme; built-ins live in the frontend and have no row here.
type Theme struct {
	ent.Schema
}

// ThemeBranding carries the whitelabel text/link overrides bundled with a
// theme. Empty fields fall back to the stock HomeBox branding.
type ThemeBranding struct {
	AppName       string       `json:"appName"`
	LoginSubtitle string       `json:"loginSubtitle"`
	SocialLinks   []SocialLink `json:"socialLinks"`
}

// SocialLink is one entry in the login page's link row.
type SocialLink struct {
	Icon  string `json:"icon"` // github | mastodon | discord | docs | link
	Label string `json:"label"`
	URL   string `json:"url"`
}

func (Theme) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.BaseMixin{},
	}
}

// Fields of the Theme.
func (Theme) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(255).
			NotEmpty(),
		// colors holds exactly the six core palette entries (background,
		// foreground, primary, secondary, accent, destructive) as #rrggbb;
		// every other CSS variable is derived from these in the frontend.
		field.JSON("colors", map[string]string{}),
		field.String("radius").
			Optional().
			Default("0.5rem"),
		// Google Font family names; empty means the system font stack.
		field.String("font_sans").
			Optional().
			Default(""),
		field.String("font_mono").
			Optional().
			Default(""),
		field.JSON("branding", ThemeBranding{}),
		// Blob storage paths for uploaded branding images; empty means the
		// stock SVG logo/icon is used.
		field.String("nav_logo_path").
			Optional().
			Default(""),
		field.String("sidebar_logo_path").
			Optional().
			Default(""),
		field.String("login_icon_path").
			Optional().
			Default(""),
	}
}
