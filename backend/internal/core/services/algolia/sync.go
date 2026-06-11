package algolia

import (
	"context"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// groupRecords builds the index records for one group under the current
// configuration.
func (m *Manager) groupRecords(ctx context.Context, gid uuid.UUID) ([]map[string]any, error) {
	rt := m.settings.Get()

	entities, err := m.repos.Entities.GetAllForIndex(ctx, gid)
	if err != nil {
		return nil, err
	}
	paths, err := m.repos.Entities.EntityPaths(ctx, gid)
	if err != nil {
		return nil, err
	}

	allow := parseFieldAllowlist(rt.Algolia.Fields)
	base := publicBaseURL(rt.Algolia, rt.Options.Hostname)

	records := make([]map[string]any, 0, len(entities))
	for _, e := range entities {
		records = append(records, buildRecord(e, gid, paths, allow, base))
	}
	return records, nil
}

// SyncGroup incrementally pushes one group's items: batch-save every current
// record, then delete index records for items that no longer exist.
func (m *Manager) SyncGroup(gid uuid.UUID) {
	client, conf := m.searchClient()
	if client == nil {
		return
	}

	m.syncMu.Lock()
	defer m.syncMu.Unlock()

	ctx := context.Background()
	records, err := m.groupRecords(ctx, gid)
	if err != nil {
		log.Error().Err(err).Str("group_id", gid.String()).Msg("algolia: failed to build records")
		return
	}

	if len(records) > 0 {
		if _, err := client.SaveObjects(conf.IndexName, records); err != nil {
			log.Error().Err(err).Str("group_id", gid.String()).Msg("algolia: failed to save records")
			return
		}
	}

	// Collect the group's objectIDs currently in the index and delete strays.
	current := make(map[string]bool, len(records))
	for _, rec := range records {
		current[rec["objectID"].(string)] = true
	}

	var stray []string
	filters := "groupId:" + gid.String()
	err = client.BrowseObjects(conf.IndexName, search.BrowseParamsObject{
		Filters:              &filters,
		AttributesToRetrieve: []string{"objectID"},
	}, search.WithAggregator(func(res any, err error) {
		if err != nil {
			return
		}
		resp, ok := res.(*search.BrowseResponse)
		if !ok {
			return
		}
		for _, hit := range resp.Hits {
			if !current[hit.ObjectID] {
				stray = append(stray, hit.ObjectID)
			}
		}
	}))
	if err != nil {
		log.Error().Err(err).Str("group_id", gid.String()).Msg("algolia: failed to browse index for cleanup")
		return
	}

	if len(stray) > 0 {
		if _, err := client.DeleteObjects(conf.IndexName, stray); err != nil {
			log.Error().Err(err).Str("group_id", gid.String()).Msg("algolia: failed to delete stray records")
			return
		}
	}

	log.Info().
		Str("group_id", gid.String()).
		Int("records", len(records)).
		Int("deleted", len(stray)).
		Msg("algolia: group synced")
}

// FullReindex atomically replaces the whole index with records from every
// group (temporary index + move), which also clears stray records in one
// shot. Used at startup, on a schedule, on settings changes, and by the
// "Reindex now" button.
func (m *Manager) FullReindex() {
	client, conf := m.searchClient()
	if client == nil {
		return
	}

	m.syncMu.Lock()
	defer m.syncMu.Unlock()

	ctx := context.Background()
	groups, err := m.repos.Groups.GetAllGroups(ctx)
	if err != nil {
		log.Error().Err(err).Msg("algolia: failed to list groups for reindex")
		return
	}

	var records []map[string]any
	for _, g := range groups {
		recs, err := m.groupRecords(ctx, g.ID)
		if err != nil {
			log.Error().Err(err).Str("group_id", g.ID.String()).Msg("algolia: failed to build records for reindex")
			return
		}
		records = append(records, recs...)
	}

	if _, err := client.ReplaceAllObjects(conf.IndexName, records, search.WithBatchSize(1000)); err != nil {
		log.Error().Err(err).Msg("algolia: full reindex failed")
		return
	}

	m.lastFull = time.Now()
	log.Info().
		Int("records", len(records)).
		Int("groups", len(groups)).
		Str("index", conf.IndexName).
		Msg("algolia: full reindex complete")
}
