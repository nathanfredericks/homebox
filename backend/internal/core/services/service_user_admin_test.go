package services

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/core/permissions"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
)

func adminCreateFactory() UserAdminCreate {
	return UserAdminCreate{
		Name:     fk.Str(10),
		Email:    fk.Email(),
		Password: strings.Repeat("a", PasswordMinLength),
	}
}

// soleSuperAdmin arranges the DB so exactly one user (returned) holds the
// Super Admin role, restoring the bootstrap user's role on cleanup.
func soleSuperAdmin(t *testing.T) repo.UserAdminOut {
	t.Helper()
	ctx := context.Background()

	superID, err := tRepos.Roles.EnsureSuperAdmin(ctx)
	require.NoError(t, err)

	usr, err := tSvc.User.AdminCreate(ctx, adminCreateFactory())
	require.NoError(t, err)
	require.NoError(t, tRepos.Roles.SetUserRoles(ctx, usr.ID, []uuid.UUID{superID}))

	// Drop the bootstrap user's super admin so usr is the last one.
	require.NoError(t, tRepos.Roles.SetUserRoles(ctx, tUser.ID, nil))
	t.Cleanup(func() {
		_ = tRepos.Roles.SetUserRoles(ctx, tUser.ID, []uuid.UUID{superID})
		_ = tRepos.Users.Delete(ctx, usr.ID)
	})

	return usr
}

func TestAdminDelete_LastSuperAdminRejected(t *testing.T) {
	ctx := context.Background()
	usr := soleSuperAdmin(t)

	err := tSvc.User.AdminDelete(ctx, usr.ID)
	require.ErrorIs(t, err, ErrLastSuperAdmin)
}

func TestDeleteSelf_LastSuperAdminRejected(t *testing.T) {
	ctx := context.Background()
	usr := soleSuperAdmin(t)

	err := tSvc.User.DeleteSelf(ctx, usr.ID)
	require.ErrorIs(t, err, ErrLastSuperAdmin)
}

func TestAdminUpdate_RemovingLastSuperAdminRoleRejected(t *testing.T) {
	ctx := context.Background()
	usr := soleSuperAdmin(t)

	_, err := tSvc.User.AdminUpdate(ctx, usr.ID, UserAdminUpdate{
		Name:    usr.Name,
		Email:   usr.Email,
		RoleIDs: nil, // strips Super Admin
	})
	require.ErrorIs(t, err, ErrLastSuperAdmin)
}

func TestAdminUpdate_RemovalAllowedWhenAnotherSuperAdminExists(t *testing.T) {
	ctx := context.Background()

	superID, err := tRepos.Roles.EnsureSuperAdmin(ctx)
	require.NoError(t, err)

	// Bootstrap user already holds Super Admin; add a second one.
	usr, err := tSvc.User.AdminCreate(ctx, adminCreateFactory())
	require.NoError(t, err)
	t.Cleanup(func() { _ = tRepos.Users.Delete(ctx, usr.ID) })
	require.NoError(t, tRepos.Roles.SetUserRoles(ctx, usr.ID, []uuid.UUID{superID}))

	out, err := tSvc.User.AdminUpdate(ctx, usr.ID, UserAdminUpdate{
		Name:    usr.Name,
		Email:   usr.Email,
		RoleIDs: nil,
	})
	require.NoError(t, err)
	assert.Empty(t, out.Roles)
}

func TestRoleService_SuperAdminImmutable(t *testing.T) {
	ctx := context.Background()

	superID, err := tRepos.Roles.EnsureSuperAdmin(ctx)
	require.NoError(t, err)

	err = tSvc.Roles.Delete(ctx, superID)
	require.ErrorIs(t, err, ErrSuperAdminImmutable)

	_, err = tSvc.Roles.Update(ctx, superID, repo.RoleUpdate{Name: "Renamed"})
	require.ErrorIs(t, err, ErrSuperAdminImmutable)
}

func TestRoleService_ValidatesSectionsAndScopes(t *testing.T) {
	ctx := context.Background()

	_, err := tSvc.Roles.Create(ctx, repo.RoleCreate{
		Name: "bogus-" + fk.Str(6),
		Permissions: []repo.RolePermissionInput{
			{Section: "not-a-section", CanView: true},
		},
	})
	require.ErrorIs(t, err, ErrInvalidPermission)

	// Site-scoped sections cannot carry a collection scope.
	colID := uuid.New()
	_, err = tSvc.Roles.Create(ctx, repo.RoleCreate{
		Name: "bogus-scope-" + fk.Str(6),
		Permissions: []repo.RolePermissionInput{
			{Section: string(permissions.SectionUsers), CollectionID: &colID, CanView: true},
		},
	})
	require.ErrorIs(t, err, ErrInvalidPermission)
}

func TestLoginOIDC_NeverAutoCreates(t *testing.T) {
	_, err := tSvc.User.LoginOIDC(context.Background(), "https://idp.example.com", "subject-"+fk.Str(8), fk.Email(), "Ghost")
	require.ErrorIs(t, err, ErrorInvalidLogin)
}
