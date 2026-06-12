package permissions

import (
	"testing"

	"github.com/google/uuid"
)

func ptr(id uuid.UUID) *uuid.UUID { return &id }

func TestSuperAdminShortCircuits(t *testing.T) {
	s := NewSet(true, nil)
	colID := uuid.New()

	for _, section := range AllSections {
		for _, action := range []Action{ActionView, ActionCreate, ActionEdit, ActionDelete} {
			if !s.Can(section, action, colID) {
				t.Fatalf("super admin denied %s on %s", actionName(action), section)
			}
		}
	}
	if !s.CanAccessCollection(colID) {
		t.Fatal("super admin denied collection access")
	}
	all, ids := s.AccessibleCollections()
	if !all || ids != nil {
		t.Fatalf("super admin should access all collections, got all=%v ids=%v", all, ids)
	}
}

func TestPerCollectionGrant(t *testing.T) {
	colA := uuid.New()
	colB := uuid.New()

	s := NewSet(false, []Grant{
		{Section: SectionItems, CollectionID: ptr(colA), Actions: ActionView | ActionEdit},
	})

	if !s.Can(SectionItems, ActionView, colA) {
		t.Fatal("expected items:view on collection A")
	}
	if !s.Can(SectionItems, ActionEdit, colA) {
		t.Fatal("expected items:edit on collection A")
	}
	if s.Can(SectionItems, ActionDelete, colA) {
		t.Fatal("items:delete should not be granted")
	}
	if s.Can(SectionItems, ActionView, colB) {
		t.Fatal("collection B should be invisible")
	}
	if s.CanAccessCollection(colB) {
		t.Fatal("collection B should not be accessible")
	}
	if !s.CanAccessCollection(colA) {
		t.Fatal("collection A should be accessible")
	}
}

func TestAllCollectionsGrant(t *testing.T) {
	colA := uuid.New()

	s := NewSet(false, []Grant{
		{Section: SectionTags, CollectionID: nil, Actions: ActionView},
	})

	if !s.Can(SectionTags, ActionView, colA) {
		t.Fatal("all-collections grant should apply to any collection")
	}
	all, _ := s.AccessibleCollections()
	if !all {
		t.Fatal("all-collections view grant should mean all collections accessible")
	}
}

func TestUnionAcrossGrants(t *testing.T) {
	colA := uuid.New()

	// Simulates two roles: one grants view everywhere, one grants edit on A.
	s := NewSet(false, []Grant{
		{Section: SectionItems, CollectionID: nil, Actions: ActionView},
		{Section: SectionItems, CollectionID: ptr(colA), Actions: ActionEdit},
	})

	if !s.Can(SectionItems, ActionEdit, colA) {
		t.Fatal("expected edit on collection A via union")
	}
	if !s.Can(SectionItems, ActionView, colA) {
		t.Fatal("expected view on collection A via all-collections grant")
	}
	if s.Can(SectionItems, ActionEdit, uuid.New()) {
		t.Fatal("edit should not leak to other collections")
	}
}

func TestSiteScopedSections(t *testing.T) {
	s := NewSet(false, []Grant{
		{Section: SectionUsers, CollectionID: nil, Actions: ActionView | ActionCreate},
	})

	if !s.Can(SectionUsers, ActionView, uuid.Nil) {
		t.Fatal("expected users:view")
	}
	if s.Can(SectionUsers, ActionDelete, uuid.Nil) {
		t.Fatal("users:delete should not be granted")
	}
	// Site-scoped grants must not make collections accessible.
	if s.CanAccessCollection(uuid.New()) {
		t.Fatal("site grants should not grant collection access")
	}
	all, ids := s.AccessibleCollections()
	if all || len(ids) != 0 {
		t.Fatalf("site grants should not grant collections, got all=%v ids=%v", all, ids)
	}
}

func TestAccessibleCollectionsList(t *testing.T) {
	colA := uuid.New()
	colB := uuid.New()

	s := NewSet(false, []Grant{
		{Section: SectionItems, CollectionID: ptr(colA), Actions: ActionView},
		{Section: SectionTags, CollectionID: ptr(colB), Actions: ActionView},
		{Section: SectionMaintenance, CollectionID: ptr(colA), Actions: ActionView},
		// Create-without-view does not make a collection accessible.
		{Section: SectionItems, CollectionID: ptr(uuid.New()), Actions: ActionCreate},
	})

	all, ids := s.AccessibleCollections()
	if all {
		t.Fatal("should not be all collections")
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 accessible collections, got %d", len(ids))
	}
}

func TestUnknownSectionIgnored(t *testing.T) {
	s := NewSet(false, []Grant{
		{Section: Section("bogus"), CollectionID: nil, Actions: ActionView},
	})
	if len(s.Grants()) != 0 {
		t.Fatal("unknown sections should be dropped")
	}
}

func TestGrantsRoundTrip(t *testing.T) {
	colA := uuid.New()
	in := []Grant{
		{Section: SectionItems, CollectionID: nil, Actions: ActionView},
		{Section: SectionItems, CollectionID: ptr(colA), Actions: ActionEdit | ActionDelete},
		{Section: SectionUsers, CollectionID: nil, Actions: ActionView},
	}
	s := NewSet(false, in)
	out := NewSet(false, s.Grants())

	if !out.Can(SectionItems, ActionView, uuid.New()) {
		t.Fatal("round trip lost all-collections items:view")
	}
	if !out.Can(SectionItems, ActionDelete, colA) {
		t.Fatal("round trip lost items:delete on collection A")
	}
	if !out.Can(SectionUsers, ActionView, uuid.Nil) {
		t.Fatal("round trip lost users:view")
	}
}

func TestAISectionIsCollectionScoped(t *testing.T) {
	if !SectionAI.IsValid() {
		t.Fatal("ai section should be valid")
	}
	if !CollectionScoped(SectionAI) {
		t.Fatal("ai section should be collection-scoped")
	}

	colA := uuid.New()
	s := NewSet(false, []Grant{
		{Section: SectionAI, CollectionID: ptr(colA), Actions: ActionView},
	})
	if !s.Can(SectionAI, ActionView, colA) {
		t.Fatal("expected ai:view on collection A")
	}
	if s.Can(SectionAI, ActionView, uuid.New()) {
		t.Fatal("other collections should be invisible")
	}
}

func TestSectionForEntity(t *testing.T) {
	if SectionForEntity(true) != SectionLocations {
		t.Fatal("location entity should map to locations")
	}
	if SectionForEntity(false) != SectionItems {
		t.Fatal("non-location entity should map to items")
	}
}

func actionName(a Action) string {
	switch a {
	case ActionView:
		return "view"
	case ActionCreate:
		return "create"
	case ActionEdit:
		return "edit"
	case ActionDelete:
		return "delete"
	}
	return "unknown"
}
