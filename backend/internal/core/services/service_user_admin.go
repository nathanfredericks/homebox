package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/sysadminsmedia/homebox/backend/pkgs/hasher"

	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
)

// Admin-facing user management. New users are created here exclusively —
// self-registration only exists for first-time setup.

type (
	UserAdminCreate struct {
		Name     string      `json:"name" validate:"required,max=255"`
		Email    string      `json:"email" validate:"required,email"`
		Password string      `json:"password" validate:"required"`
		RoleIDs  []uuid.UUID `json:"roleIds"`
	}

	UserAdminUpdate struct {
		Name  string `json:"name" validate:"required,max=255"`
		Email string `json:"email" validate:"required,email"`
		// Password optionally resets the user's password when non-empty.
		Password string      `json:"password"`
		RoleIDs  []uuid.UUID `json:"roleIds"`
	}
)

func (svc *UserService) AdminList(ctx context.Context) ([]repo.UserAdminOut, error) {
	return svc.repos.Users.GetAll(ctx)
}

func (svc *UserService) AdminGet(ctx context.Context, id uuid.UUID) (repo.UserAdminOut, error) {
	usr, err := svc.repos.Users.GetOneID(ctx, id)
	if err != nil {
		return repo.UserAdminOut{}, err
	}
	roles, err := svc.repos.Roles.GetUserRoles(ctx, id)
	if err != nil {
		return repo.UserAdminOut{}, err
	}
	return repo.UserAdminOut{UserOut: usr, Roles: roles}, nil
}

func (svc *UserService) AdminCreate(ctx context.Context, data UserAdminCreate) (repo.UserAdminOut, error) {
	if len(data.Password) < PasswordMinLength {
		return repo.UserAdminOut{}, ErrorPasswordTooShort
	}

	hashed, err := hasher.HashPasswordCtx(ctx, data.Password)
	if err != nil {
		return repo.UserAdminOut{}, err
	}

	usr, err := svc.repos.Users.Create(ctx, repo.UserCreate{
		Name:     data.Name,
		Email:    data.Email,
		Password: &hashed,
	})
	if err != nil {
		return repo.UserAdminOut{}, err
	}

	if len(data.RoleIDs) > 0 {
		if err := svc.repos.Roles.SetUserRoles(ctx, usr.ID, data.RoleIDs); err != nil {
			return repo.UserAdminOut{}, err
		}
	}

	log.Info().Str("user_id", usr.ID.String()).Msg("admin created user")
	return svc.AdminGet(ctx, usr.ID)
}

func (svc *UserService) AdminUpdate(ctx context.Context, id uuid.UUID, data UserAdminUpdate) (repo.UserAdminOut, error) {
	// Enforce the Super Admin invariant before touching anything: removing
	// the last super admin's role assignment is rejected regardless of who
	// asks — which also prevents a super admin demoting themselves unless
	// another super admin exists.
	hasSuper, err := svc.repos.Roles.UserHasSuperAdmin(ctx, id)
	if err != nil {
		return repo.UserAdminOut{}, err
	}
	if hasSuper {
		keepsSuper := false
		for _, rid := range data.RoleIDs {
			if has, err := svc.roleIsSuperAdmin(ctx, rid); err != nil {
				return repo.UserAdminOut{}, err
			} else if has {
				keepsSuper = true
				break
			}
		}
		if !keepsSuper {
			count, err := svc.repos.Roles.CountSuperAdminUsers(ctx)
			if err != nil {
				return repo.UserAdminOut{}, err
			}
			if count <= 1 {
				return repo.UserAdminOut{}, ErrLastSuperAdmin
			}
		}
	}

	if err := svc.repos.Users.Update(ctx, id, repo.UserUpdate{Name: data.Name, Email: data.Email}); err != nil {
		return repo.UserAdminOut{}, err
	}

	if data.Password != "" {
		if len(data.Password) < PasswordMinLength {
			return repo.UserAdminOut{}, ErrorPasswordTooShort
		}
		hashed, err := hasher.HashPasswordCtx(ctx, data.Password)
		if err != nil {
			return repo.UserAdminOut{}, err
		}
		if err := svc.repos.Users.ChangePassword(ctx, id, hashed); err != nil {
			return repo.UserAdminOut{}, err
		}
	}

	if err := svc.repos.Roles.SetUserRoles(ctx, id, data.RoleIDs); err != nil {
		return repo.UserAdminOut{}, err
	}

	return svc.AdminGet(ctx, id)
}

// AdminDelete removes a user. Deleting the last super admin is rejected.
func (svc *UserService) AdminDelete(ctx context.Context, id uuid.UUID) error {
	if err := svc.guardLastSuperAdmin(ctx, id); err != nil {
		return err
	}
	return svc.repos.Users.Delete(ctx, id)
}

// guardLastSuperAdmin rejects operations that would remove the last super
// admin from the system (delete, self-delete).
func (svc *UserService) guardLastSuperAdmin(ctx context.Context, userID uuid.UUID) error {
	hasSuper, err := svc.repos.Roles.UserHasSuperAdmin(ctx, userID)
	if err != nil {
		return err
	}
	if !hasSuper {
		return nil
	}
	count, err := svc.repos.Roles.CountSuperAdminUsers(ctx)
	if err != nil {
		return err
	}
	if count <= 1 {
		return ErrLastSuperAdmin
	}
	return nil
}

func (svc *UserService) roleIsSuperAdmin(ctx context.Context, roleID uuid.UUID) (bool, error) {
	r, err := svc.repos.Roles.GetOneID(ctx, roleID)
	if err != nil {
		return false, err
	}
	return r.IsSuperAdmin, nil
}
