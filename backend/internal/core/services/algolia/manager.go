package algolia

import (
	"sync"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/settings"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
)

// mutationDebounce is how long a group's sync waits after the last mutation,
// coalescing bursts (imports, bulk edits) into one push.
const mutationDebounce = 10 * time.Second

// Manager owns the Algolia client lifecycle and the sync triggers. All public
// methods are safe for concurrent use and never return errors: indexing is
// best-effort and must not affect inventory operations.
type Manager struct {
	settings *settings.Service
	repos    *repo.AllRepos

	mu         sync.Mutex
	client     *search.APIClient
	clientConf config.AlgoliaConf // conf the cached client was built for
	timers     map[uuid.UUID]*time.Timer

	// syncMu serializes pushes so a full reindex and group syncs don't
	// interleave their batch operations. lastFull is guarded by it.
	syncMu   sync.Mutex
	lastFull time.Time
}

func NewManager(s *settings.Service, repos *repo.AllRepos) *Manager {
	m := &Manager{
		settings: s,
		repos:    repos,
		timers:   map[uuid.UUID]*time.Timer{},
	}

	s.OnChange(func(old, new settings.Resolved) {
		if old.Algolia == new.Algolia && old.Options.Hostname == new.Options.Hostname {
			return
		}

		m.mu.Lock()
		m.client = nil // rebuilt lazily against the new configuration
		m.mu.Unlock()

		// Any material change while enabled warrants a full rebuild so the
		// index matches the new record shape / destination.
		if new.Algolia.Enabled {
			log.Info().Msg("algolia: settings changed, scheduling full reindex")
			go m.FullReindex()
		}
	})

	return m
}

// Subscribe registers the mutation listeners. Tag and import events are
// included because they change record content without an entity mutation.
func (m *Manager) Subscribe(bus *eventbus.EventBus) {
	bus.Subscribe(eventbus.EventEntityMutation, m.OnMutation)
	bus.Subscribe(eventbus.EventTagMutation, m.OnMutation)
	bus.Subscribe(eventbus.EventImportMutation, m.OnMutation)
}

// OnMutation debounces a group sync. The eventbus payload only carries the
// group ID, so the whole group is re-pushed.
func (m *Manager) OnMutation(data any) {
	event, ok := data.(eventbus.GroupMutationEvent)
	if !ok {
		return
	}
	if !m.settings.Get().Algolia.Enabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if t, ok := m.timers[event.GID]; ok {
		t.Reset(mutationDebounce)
		return
	}
	gid := event.GID
	m.timers[gid] = time.AfterFunc(mutationDebounce, func() {
		m.mu.Lock()
		delete(m.timers, gid)
		m.mu.Unlock()
		m.SyncGroup(gid)
	})
}

// searchClient returns a client for the current configuration, rebuilding it
// when the configuration changed. Returns nil (after logging once per rebuild
// attempt) when the integration is disabled or incomplete.
func (m *Manager) searchClient() (*search.APIClient, config.AlgoliaConf) {
	conf := m.settings.Get().Algolia

	m.mu.Lock()
	defer m.mu.Unlock()

	if !conf.Enabled || conf.AppID == "" || conf.AdminAPIKey == "" || conf.IndexName == "" {
		return nil, conf
	}
	if m.client != nil && m.clientConf == conf {
		return m.client, conf
	}

	client, err := search.NewClient(conf.AppID, conf.AdminAPIKey)
	if err != nil {
		log.Error().Err(err).Msg("algolia: failed to build client")
		return nil, conf
	}

	// groupId must be a filter-only facet for tenant-scoped queries and for
	// the stray-record cleanup in SyncGroup.
	_, err = client.SetSettings(client.NewApiSetSettingsRequest(conf.IndexName, &search.IndexSettings{
		AttributesForFaceting: []string{"filterOnly(groupId)"},
	}))
	if err != nil {
		log.Error().Err(err).Msg("algolia: failed to apply index settings")
	}

	m.client = client
	m.clientConf = conf
	return m.client, conf
}

// ReindexInterval returns the configured full-reindex period, defaulting to
// 24h when the setting is empty or unparsable.
func (m *Manager) ReindexInterval() time.Duration {
	raw := m.settings.Get().Algolia.ReindexInterval
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 24 * time.Hour
	}
	return d
}

// Enabled reports whether the integration is currently switched on.
func (m *Manager) Enabled() bool {
	return m.settings.Get().Algolia.Enabled
}

// LastFullReindex returns when the last successful full reindex finished
// (zero time if none has run yet), for the periodic scheduler.
func (m *Manager) LastFullReindex() time.Time {
	m.syncMu.Lock()
	defer m.syncMu.Unlock()
	return m.lastFull
}
