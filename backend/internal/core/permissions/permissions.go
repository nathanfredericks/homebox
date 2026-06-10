// Package permissions defines the granular RBAC model: sections mirror the
// frontend UI surfaces, each grantable with View/Create/Edit/Delete actions,
// scoped either site-wide or per-collection. A user's effective permission
// Set is the union of all grants across their roles; the Super Admin role
// short-circuits every check.
package permissions

import (
	"slices"

	"github.com/google/uuid"
)

// Section identifies one UI surface that permissions are granted against.
type Section string

const (
	// Inventory surfaces (collection-scoped).
	SectionItems       Section = "items"
	SectionLocations   Section = "locations"
	SectionTags        Section = "tags"
	SectionTemplates   Section = "templates"
	SectionMaintenance Section = "maintenance"
	SectionStatistics  Section = "statistics"

	// Collection settings surfaces (collection-scoped).
	SectionCollectionSettings Section = "collection_settings"
	SectionEntityTypes        Section = "entity_types"
	SectionNotifiers          Section = "notifiers"
	SectionTools              Section = "tools"

	// Administration surfaces (site-scoped).
	SectionUsers       Section = "users"
	SectionRoles       Section = "roles"
	SectionCollections Section = "collections"
)

// AllSections lists every valid section, in UI display order.
var AllSections = []Section{
	SectionItems,
	SectionLocations,
	SectionTags,
	SectionTemplates,
	SectionMaintenance,
	SectionStatistics,
	SectionCollectionSettings,
	SectionEntityTypes,
	SectionNotifiers,
	SectionTools,
	SectionUsers,
	SectionRoles,
	SectionCollections,
}

var siteSections = map[Section]bool{
	SectionUsers:       true,
	SectionRoles:       true,
	SectionCollections: true,
}

// IsValid reports whether s is a known section.
func (s Section) IsValid() bool {
	return slices.Contains(AllSections, s)
}

// CollectionScoped reports whether grants for this section apply to a
// collection (or all collections) rather than to the site.
func CollectionScoped(s Section) bool {
	return !siteSections[s]
}

// SectionForEntity maps an entity to its permission section based on whether
// its entity type is a location type.
func SectionForEntity(isLocation bool) Section {
	if isLocation {
		return SectionLocations
	}
	return SectionItems
}

// Action is a bitmask of the four basic actions.
type Action uint8

const (
	ActionView Action = 1 << iota
	ActionCreate
	ActionEdit
	ActionDelete
)

// Grant is one serializable permission row: a section, an optional collection
// scope (nil = all collections, or the site scope for site sections), and the
// granted actions.
type Grant struct {
	Section      Section
	CollectionID *uuid.UUID
	Actions      Action
}

// Set is a user's effective permissions, assembled as the union of all grant
// rows across their roles.
type Set struct {
	SuperAdmin bool
	// grants[section][collectionID] -> actions; uuid.Nil keys hold
	// all-collections grants and site-scope grants.
	grants map[Section]map[uuid.UUID]Action
}

// NewSet builds a Set from grant rows. Rows with unknown sections are ignored.
func NewSet(superAdmin bool, grants []Grant) *Set {
	s := &Set{
		SuperAdmin: superAdmin,
		grants:     map[Section]map[uuid.UUID]Action{},
	}
	for _, g := range grants {
		if !g.Section.IsValid() {
			continue
		}
		key := uuid.Nil
		if g.CollectionID != nil {
			key = *g.CollectionID
		}
		if s.grants[g.Section] == nil {
			s.grants[g.Section] = map[uuid.UUID]Action{}
		}
		s.grants[g.Section][key] |= g.Actions
	}
	return s
}

// Can reports whether the action is granted for the section. For
// collection-scoped sections pass the collection ID; both an all-collections
// grant (uuid.Nil) and a collection-specific grant satisfy the check. For
// site-scoped sections pass uuid.Nil.
func (s *Set) Can(section Section, action Action, collectionID uuid.UUID) bool {
	if s.SuperAdmin {
		return true
	}
	scoped := s.grants[section]
	if scoped == nil {
		return false
	}
	granted := scoped[uuid.Nil]
	if collectionID != uuid.Nil {
		granted |= scoped[collectionID]
	}
	return granted&action == action
}

// CanAccessCollection reports whether the user can enter the collection at
// all: any collection-scoped section with View access (all-collections or
// this collection) qualifies.
func (s *Set) CanAccessCollection(id uuid.UUID) bool {
	if s.SuperAdmin {
		return true
	}
	for section, scoped := range s.grants {
		if !CollectionScoped(section) {
			continue
		}
		if (scoped[uuid.Nil]|scoped[id])&ActionView == ActionView {
			return true
		}
	}
	return false
}

// AccessibleCollections returns the collections the user can access. If all
// is true the user holds an all-collections view grant (or is a Super Admin)
// and ids is nil; otherwise ids lists the explicitly granted collections.
func (s *Set) AccessibleCollections() (all bool, ids []uuid.UUID) {
	if s.SuperAdmin {
		return true, nil
	}
	seen := map[uuid.UUID]bool{}
	for section, scoped := range s.grants {
		if !CollectionScoped(section) {
			continue
		}
		for id, actions := range scoped {
			if actions&ActionView != ActionView {
				continue
			}
			if id == uuid.Nil {
				return true, nil
			}
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	return false, ids
}

// Grants serializes the set back into rows (one per section+scope), for the
// /users/self payload.
func (s *Set) Grants() []Grant {
	out := []Grant{}
	for _, section := range AllSections {
		scoped := s.grants[section]
		// Deterministic order: all-collections row first, then specific ones.
		if actions, ok := scoped[uuid.Nil]; ok && actions != 0 {
			out = append(out, Grant{Section: section, CollectionID: nil, Actions: actions})
		}
		for id, actions := range scoped {
			if id == uuid.Nil || actions == 0 {
				continue
			}
			cid := id
			out = append(out, Grant{Section: section, CollectionID: &cid, Actions: actions})
		}
	}
	return out
}
