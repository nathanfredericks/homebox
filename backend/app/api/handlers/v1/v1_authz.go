package v1

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/core/permissions"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
)

// Entity routes are shared between Items and Locations (this fork models
// locations as entities whose type has is_location). Route middleware
// fail-fasts with mwPermissionAny; these helpers resolve the entity's real
// section and enforce the precise permission.
//
// Per the invisibility rule: missing View → 404 (the entity does not exist
// for this user); missing write action on a viewable entity → 403.

var (
	errNotFound  = validate.NewRequestError(errors.New("not found"), http.StatusNotFound)
	errForbidden = validate.NewRequestError(errors.New("forbidden"), http.StatusForbidden)
)

// checkEntityPermission enforces action on the section matching the entity's
// kind (items or locations) within the request tenant.
func (ctrl *V1Controller) checkEntityPermission(r *http.Request, entityID uuid.UUID, action permissions.Action) error {
	auth := services.NewContext(r.Context())

	isLocation, err := ctrl.repo.Entities.IsLocation(auth, auth.GID, entityID)
	if err != nil {
		if ent.IsNotFound(err) {
			return errNotFound
		}
		return err
	}

	return checkSectionAction(auth.Perms, permissions.SectionForEntity(isLocation), action, auth.GID)
}

// checkEntityTypePermission enforces action on the section matching the kind
// of the given entity type (used for create flows, where the resulting kind
// is decided by the type). A nil typeID resolves to items (the default type).
func (ctrl *V1Controller) checkEntityTypePermission(r *http.Request, typeID uuid.UUID, action permissions.Action) error {
	auth := services.NewContext(r.Context())

	isLocation := false
	if typeID != uuid.Nil {
		var err error
		isLocation, err = ctrl.repo.Entities.TypeIsLocation(auth, auth.GID, typeID)
		if err != nil {
			if ent.IsNotFound(err) {
				return errNotFound
			}
			return err
		}
	}

	return checkSectionAction(auth.Perms, permissions.SectionForEntity(isLocation), action, auth.GID)
}

func checkSectionAction(set *permissions.Set, section permissions.Section, action permissions.Action, gid uuid.UUID) error {
	if set.Can(section, action, gid) {
		return nil
	}
	if !set.Can(section, permissions.ActionView, gid) {
		return errNotFound
	}
	return errForbidden
}

// clampEntityKindFilter restricts an entity list query to the kinds the user
// can see in the tenant. Returns false when neither kind is visible (the
// route middleware should have caught this already).
func clampEntityKindFilter(set *permissions.Set, gid uuid.UUID, isLocation *bool) (*bool, bool) {
	canItems := set.Can(permissions.SectionItems, permissions.ActionView, gid)
	canLocations := set.Can(permissions.SectionLocations, permissions.ActionView, gid)

	switch {
	case canItems && canLocations:
		return isLocation, true
	case canItems:
		f := false
		if isLocation != nil && *isLocation {
			return nil, false
		}
		return &f, true
	case canLocations:
		t := true
		if isLocation != nil && !*isLocation {
			return nil, false
		}
		return &t, true
	default:
		return nil, false
	}
}
