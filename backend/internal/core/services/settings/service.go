package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
)

// RedactedSentinel is what secret fields read as through the API. Writing it
// back keeps the currently effective secret.
const RedactedSentinel = "[REDACTED]"

var (
	// ErrUnknownSection is returned for section names outside SectionNames.
	ErrUnknownSection = errors.New("settings: unknown section")
	// ErrInvalidPayload wraps JSON payloads that don't fit the section struct.
	ErrInvalidPayload = errors.New("settings: invalid payload")
)

// MutationEvent is the payload published on eventbus.EventSettingsMutation.
type MutationEvent struct {
	Section string
}

type state struct {
	resolved Resolved
	// overridden marks sections that currently have a database row.
	overridden map[string]bool
}

// Service layers database overrides over the startup configuration and hands
// out the effective values. Reads are lock-free snapshots; writes rebuild the
// snapshot and notify listeners.
type Service struct {
	repo *repo.SiteSettingsRepository
	bus  *eventbus.EventBus
	base Resolved

	mu        sync.Mutex // serializes writes and cache rebuilds
	cache     atomic.Pointer[state]
	listeners []func(old, new Resolved)
}

// New builds the service and eagerly resolves the current state so a broken
// database surfaces at startup rather than on first read.
func New(ctx context.Context, r *repo.SiteSettingsRepository, cfg *config.Config, bus *eventbus.EventBus) (*Service, error) {
	s := &Service{
		repo: r,
		bus:  bus,
		base: Resolved{
			Options:    cfg.Options,
			Thumbnail:  cfg.Thumbnail,
			Barcode:    cfg.Barcode,
			Mailer:     cfg.Mailer,
			LabelMaker: cfg.LabelMaker,
			Notifier:   cfg.Notifier,
			Algolia:    cfg.Algolia,
		},
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.rebuildLocked(ctx); err != nil {
		return nil, fmt.Errorf("settings: resolving initial state: %w", err)
	}
	return s, nil
}

// Get returns the effective configuration snapshot.
func (s *Service) Get() Resolved {
	return s.cache.Load().resolved
}

// Overridden reports which sections currently have a database override.
func (s *Service) Overridden() map[string]bool {
	cur := s.cache.Load().overridden
	out := make(map[string]bool, len(cur))
	for k, v := range cur {
		out[k] = v
	}
	return out
}

// OnChange registers a listener invoked (synchronously, in registration
// order) after every successful update or reset. Listeners doing slow work
// must spawn their own goroutines.
func (s *Service) OnChange(fn func(old, new Resolved)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, fn)
}

// UpdateSection validates and persists a sparse override document for one
// section, merged over any keys already overridden so callers only need to
// send what changed. Secret fields carrying the redaction sentinel keep their
// currently effective value; env-only fields are stripped. Removing a single
// key is not possible — ResetSection drops the whole section.
func (s *Service) UpdateSection(ctx context.Context, section string, payload json.RawMessage) error {
	if !slices.Contains(SectionNames, section) {
		return ErrUnknownSection
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cleaned, err := s.cleanPayloadLocked(section, payload)
	if err != nil {
		return err
	}

	merged, err := s.mergeWithStoredLocked(ctx, section, cleaned)
	if err != nil {
		return err
	}

	old := s.cache.Load().resolved
	if err := s.repo.Upsert(ctx, section, merged); err != nil {
		return err
	}
	if err := s.rebuildLocked(ctx); err != nil {
		return err
	}
	s.notifyLocked(old, section)
	return nil
}

// ResetSection removes a section's database override, restoring the
// environment/default values.
func (s *Service) ResetSection(ctx context.Context, section string) error {
	if !slices.Contains(SectionNames, section) {
		return ErrUnknownSection
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	old := s.cache.Load().resolved
	if err := s.repo.Delete(ctx, section); err != nil {
		return err
	}
	if err := s.rebuildLocked(ctx); err != nil {
		return err
	}
	s.notifyLocked(old, section)
	return nil
}

// cleanPayloadLocked decodes the incoming document as a flat JSON object,
// rejects unknown or env-only keys, substitutes secret sentinels with the
// currently effective values, and verifies the result unmarshals into the
// section struct.
func (s *Service) cleanPayloadLocked(section string, payload json.RawMessage) (json.RawMessage, error) {
	var doc map[string]json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidPayload, err)
	}

	for _, key := range sectionEnvOnly[section] {
		delete(doc, key)
	}

	cur := s.cache.Load().resolved
	for _, key := range sectionSecrets[section] {
		raw, ok := doc[key]
		if !ok {
			continue
		}
		var v string
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("%w: field %q must be a string", ErrInvalidPayload, key)
		}
		if v == RedactedSentinel {
			replaced, _ := json.Marshal(currentSecret(&cur, section, key))
			doc[key] = replaced
		}
	}

	cleaned, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}

	// Strict round-trip into the section struct catches unknown keys and type
	// mismatches before anything reaches the database.
	scratch := s.base
	dec = json.NewDecoder(bytes.NewReader(cleaned))
	dec.DisallowUnknownFields()
	if err := dec.Decode(sectionPtr(&scratch, section)); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidPayload, err)
	}

	return cleaned, nil
}

// mergeWithStoredLocked overlays the cleaned payload's keys onto the
// section's existing override document, if any. Callers hold s.mu.
func (s *Service) mergeWithStoredLocked(ctx context.Context, section string, cleaned json.RawMessage) (json.RawMessage, error) {
	rows, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	existing, ok := rows[section]
	if !ok {
		return cleaned, nil
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(existing, &doc); err != nil {
		// A corrupt stored row should not block recovery via update.
		return cleaned, nil
	}
	var incoming map[string]json.RawMessage
	if err := json.Unmarshal(cleaned, &incoming); err != nil {
		return nil, err
	}
	for k, v := range incoming {
		doc[k] = v
	}
	return json.Marshal(doc)
}

// rebuildLocked re-resolves the snapshot from base + database rows. Callers
// hold s.mu.
func (s *Service) rebuildLocked(ctx context.Context) error {
	rows, err := s.repo.GetAll(ctx)
	if err != nil {
		return err
	}

	next := state{
		resolved:   s.base,
		overridden: make(map[string]bool, len(rows)),
	}
	for _, name := range SectionNames {
		raw, ok := rows[name]
		if !ok {
			continue
		}
		// A row is a sparse override: unmarshal only touches present keys, so
		// everything else keeps its environment/default value. Rows that fail
		// to decode are skipped rather than taking the whole service down.
		if err := json.Unmarshal(raw, sectionPtr(&next.resolved, name)); err != nil {
			return fmt.Errorf("settings: decoding section %q: %w", name, err)
		}
		next.overridden[name] = true
	}

	s.cache.Store(&next)
	return nil
}

// notifyLocked fires change listeners and the settings mutation event.
// Callers hold s.mu.
func (s *Service) notifyLocked(old Resolved, section string) {
	cur := s.cache.Load().resolved
	for _, fn := range s.listeners {
		fn(old, cur)
	}
	if s.bus != nil {
		s.bus.Publish(eventbus.EventSettingsMutation, MutationEvent{Section: section})
	}
}
