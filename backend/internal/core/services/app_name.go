package services

import (
	"context"

	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
)

const defaultAppName = "Homebox"

// activeAppName returns the whitelabel app name from the active theme's
// branding, falling back to the stock name when no branding is configured.
func activeAppName(ctx context.Context, repos *repo.AllRepos) string {
	_, theme, err := repos.Themes.GetActiveTheme(ctx)
	if err != nil || theme == nil || theme.Branding.AppName == "" {
		return defaultAppName
	}
	return theme.Branding.AppName
}
