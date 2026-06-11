package main

import (
	"time"

	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/algolia"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/settings"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/otel"
)

type app struct {
	conf                *config.Config
	settings            *settings.Service
	algolia             *algolia.Manager
	db                  *ent.Client
	repos               *repo.AllRepos
	services            *services.AllServices
	bus                 *eventbus.EventBus
	authLimiter         *authRateLimiter
	notifierTestLimiter *simpleRateLimiter
	otel                *otel.Provider
}

func new(conf *config.Config) *app {
	s := &app{
		conf: conf,
	}

	s.authLimiter = newAuthRateLimiter(s.conf.Auth.RateLimit)
	s.notifierTestLimiter = newSimpleRateLimiter(10, time.Minute, s.conf.Options.TrustProxy) // 10 requests per minute

	return s
}
