package repo

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/schema"
)

func themeFactory() ThemeCreate {
	return ThemeCreate{
		Name: fk.Str(10),
		Colors: map[string]string{
			"background":  "#ffffff",
			"foreground":  "#333333",
			"primary":     "#5b7f67",
			"secondary":   "#2c2f28",
			"accent":      "#e7f2e3",
			"destructive": "#f87171",
		},
		Radius:   "0.5rem",
		FontSans: "Inter",
		FontMono: "",
		Branding: schema.ThemeBranding{
			AppName:       "Acme Inventory",
			LoginSubtitle: "Track Acme things",
			SocialLinks: []schema.SocialLink{
				{Icon: "link", Label: "Acme", URL: "https://acme.example"},
			},
		},
	}
}

func TestThemeRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	created, err := tRepos.Themes.Create(ctx, themeFactory())
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, created.ID)
	assert.Equal(t, "Acme Inventory", created.Branding.AppName)
	assert.Equal(t, "#5b7f67", created.Colors["primary"])
	assert.False(t, created.Assets.NavLogo)

	got, err := tRepos.Themes.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.Name, got.Name)

	all, err := tRepos.Themes.GetAll(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, all)

	update := ThemeUpdate(themeFactory())
	update.Name = "renamed"
	update.Colors["primary"] = "#112233"
	updated, err := tRepos.Themes.Update(ctx, created.ID, update)
	require.NoError(t, err)
	assert.Equal(t, "renamed", updated.Name)
	assert.Equal(t, "#112233", updated.Colors["primary"])

	require.NoError(t, tRepos.Themes.Delete(ctx, created.ID))
	_, err = tRepos.Themes.Get(ctx, created.ID)
	require.Error(t, err)
}

func TestThemeRepository_ColorValidation(t *testing.T) {
	ctx := context.Background()

	bad := themeFactory()
	bad.Colors["primary"] = "not-a-color"
	_, err := tRepos.Themes.Create(ctx, bad)
	assert.ErrorIs(t, err, ErrThemeInvalidColors)

	missing := themeFactory()
	delete(missing.Colors, "accent")
	_, err = tRepos.Themes.Create(ctx, missing)
	assert.ErrorIs(t, err, ErrThemeInvalidColors)

	extra := themeFactory()
	extra.Colors["card"] = "#ffffff"
	_, err = tRepos.Themes.Create(ctx, extra)
	assert.ErrorIs(t, err, ErrThemeInvalidColors)
}

func TestThemeRepository_ActivePointer(t *testing.T) {
	ctx := context.Background()

	// Default when nothing is stored.
	active, err := tRepos.Themes.GetActive(ctx)
	require.NoError(t, err)
	assert.Equal(t, DefaultActiveTheme, active)

	// Built-in activation.
	require.NoError(t, tRepos.Themes.SetActive(ctx, "builtin:dracula"))
	active, err = tRepos.Themes.GetActive(ctx)
	require.NoError(t, err)
	assert.Equal(t, "builtin:dracula", active)

	// Custom activation requires an existing theme.
	assert.ErrorIs(t, tRepos.Themes.SetActive(ctx, "custom:"+uuid.NewString()), ErrThemeInvalidPointer)
	assert.ErrorIs(t, tRepos.Themes.SetActive(ctx, "garbage"), ErrThemeInvalidPointer)
	assert.ErrorIs(t, tRepos.Themes.SetActive(ctx, "builtin:"), ErrThemeInvalidPointer)

	created, err := tRepos.Themes.Create(ctx, themeFactory())
	require.NoError(t, err)
	require.NoError(t, tRepos.Themes.SetActive(ctx, "custom:"+created.ID.String()))

	pointer, themeOut, err := tRepos.Themes.GetActiveTheme(ctx)
	require.NoError(t, err)
	assert.Equal(t, "custom:"+created.ID.String(), pointer)
	require.NotNil(t, themeOut)
	assert.Equal(t, created.ID, themeOut.ID)

	// The active theme cannot be deleted.
	assert.ErrorIs(t, tRepos.Themes.Delete(ctx, created.ID), ErrThemeActive)

	// Reset and clean up.
	require.NoError(t, tRepos.Themes.SetActive(ctx, DefaultActiveTheme))
	require.NoError(t, tRepos.Themes.Delete(ctx, created.ID))

	pointer, themeOut, err = tRepos.Themes.GetActiveTheme(ctx)
	require.NoError(t, err)
	assert.Equal(t, DefaultActiveTheme, pointer)
	assert.Nil(t, themeOut)
}

func TestThemeRepository_Assets(t *testing.T) {
	ctx := context.Background()

	created, err := tRepos.Themes.Create(ctx, themeFactory())
	require.NoError(t, err)
	defer func() { _ = tRepos.Themes.Delete(ctx, created.ID) }()

	content := []byte("<svg xmlns='http://www.w3.org/2000/svg'/>")
	out, err := tRepos.Themes.SetAsset(ctx, created.ID, "nav-logo", "logo.svg", bytes.NewReader(content))
	require.NoError(t, err)
	assert.True(t, out.Assets.NavLogo)
	assert.False(t, out.Assets.SidebarLogo)

	path, err := tRepos.Themes.AssetPath(ctx, created.ID, "nav-logo")
	require.NoError(t, err)
	assert.NotEmpty(t, path)

	// Inactive theme exposes no public asset.
	activePath, err := tRepos.Themes.ActiveAssetPath(ctx, "nav-logo")
	require.NoError(t, err)
	assert.Empty(t, activePath)

	// Activating the theme exposes it.
	require.NoError(t, tRepos.Themes.SetActive(ctx, "custom:"+created.ID.String()))
	activePath, err = tRepos.Themes.ActiveAssetPath(ctx, "nav-logo")
	require.NoError(t, err)
	assert.Equal(t, path, activePath)
	require.NoError(t, tRepos.Themes.SetActive(ctx, DefaultActiveTheme))

	_, err = tRepos.Themes.AssetPath(ctx, created.ID, "bogus")
	assert.ErrorIs(t, err, ErrThemeUnknownAssetKind)

	out, err = tRepos.Themes.DeleteAsset(ctx, created.ID, "nav-logo")
	require.NoError(t, err)
	assert.False(t, out.Assets.NavLogo)
}
