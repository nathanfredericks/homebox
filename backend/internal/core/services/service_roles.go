package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/core/permissions"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
)

var (
	// ErrSuperAdminImmutable is returned for attempts to edit or delete the
	// seeded Super Admin role.
	ErrSuperAdminImmutable = errors.New("the Super Admin group cannot be modified or deleted")
	// ErrLastSuperAdmin is returned when an operation would leave the system
	// without any super admin.
	ErrLastSuperAdmin = errors.New("at least one user must keep the Super Admin group")
	// ErrInvalidPermission is returned when a permission row references an
	// unknown section or an invalid scope for its section.
	ErrInvalidPermission = errors.New("invalid permission section or scope")
)

// RoleService manages roles (shown as "Groups" in the UI) and enforces the
// Super Admin invariants.
type RoleService struct {
	repos *repo.AllRepos
}

func (svc *RoleService) validatePermissions(perms []repo.RolePermissionInput) error {
	for _, p := range perms {
		section := permissions.Section(p.Section)
		if !section.IsValid() {
			return ErrInvalidPermission
		}
		// Site-scoped sections cannot be granted per-collection.
		if !permissions.CollectionScoped(section) && p.CollectionID != nil {
			return ErrInvalidPermission
		}
	}
	return nil
}

func (svc *RoleService) GetAll(ctx context.Context) ([]repo.RoleOut, error) {
	return svc.repos.Roles.GetAll(ctx)
}

func (svc *RoleService) GetOne(ctx context.Context, id uuid.UUID) (repo.RoleOut, error) {
	return svc.repos.Roles.GetOneID(ctx, id)
}

func (svc *RoleService) Create(ctx context.Context, data repo.RoleCreate) (repo.RoleOut, error) {
	if err := svc.validatePermissions(data.Permissions); err != nil {
		return repo.RoleOut{}, err
	}
	return svc.repos.Roles.Create(ctx, data)
}

func (svc *RoleService) Update(ctx context.Context, id uuid.UUID, data repo.RoleUpdate) (repo.RoleOut, error) {
	existing, err := svc.repos.Roles.GetOneID(ctx, id)
	if err != nil {
		return repo.RoleOut{}, err
	}
	if existing.IsSuperAdmin {
		return repo.RoleOut{}, ErrSuperAdminImmutable
	}
	if err := svc.validatePermissions(data.Permissions); err != nil {
		return repo.RoleOut{}, err
	}
	return svc.repos.Roles.Update(ctx, id, data)
}

func (svc *RoleService) Delete(ctx context.Context, id uuid.UUID) error {
	existing, err := svc.repos.Roles.GetOneID(ctx, id)
	if err != nil {
		return err
	}
	if existing.IsSuperAdmin {
		return ErrSuperAdminImmutable
	}
	return svc.repos.Roles.Delete(ctx, id)
}
