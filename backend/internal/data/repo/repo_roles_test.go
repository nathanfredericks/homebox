package repo

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/core/permissions"
)

func roleFactory(perms ...RolePermissionInput) RoleCreate {
	return RoleCreate{
		Name:        "role-" + fk.Str(10),
		Description: fk.Str(20),
		Permissions: perms,
	}
}

func TestRoleRepo_CRUD(t *testing.T) {
	ctx := context.Background()

	created, err := tRepos.Roles.Create(ctx, roleFactory(RolePermissionInput{
		Section: string(permissions.SectionItems),
		CanView: true,
		CanEdit: true,
	}))
	require.NoError(t, err)
	assert.Len(t, created.Permissions, 1)

	// Update replaces the permission set wholesale.
	updated, err := tRepos.Roles.Update(ctx, created.ID, RoleUpdate{
		Name:        created.Name,
		Description: created.Description,
		Permissions: []RolePermissionInput{
			{Section: string(permissions.SectionTags), CanView: true},
			{Section: string(permissions.SectionItems), CanView: true, CanDelete: true},
		},
	})
	require.NoError(t, err)
	assert.Len(t, updated.Permissions, 2)

	require.NoError(t, tRepos.Roles.Delete(ctx, created.ID))
	_, err = tRepos.Roles.GetOneID(ctx, created.ID)
	require.Error(t, err)
}

func TestRoleRepo_UserPermissionSetUnion(t *testing.T) {
	ctx := context.Background()

	collection, err := tRepos.Groups.GroupCreate(ctx, "perm-union")
	require.NoError(t, err)

	viewEverywhere, err := tRepos.Roles.Create(ctx, roleFactory(RolePermissionInput{
		Section: string(permissions.SectionItems),
		CanView: true,
	}))
	require.NoError(t, err)

	editOne, err := tRepos.Roles.Create(ctx, roleFactory(RolePermissionInput{
		Section:      string(permissions.SectionItems),
		CollectionID: &collection.ID,
		CanEdit:      true,
	}))
	require.NoError(t, err)

	usr, err := tRepos.Users.Create(ctx, userFactory())
	require.NoError(t, err)
	t.Cleanup(func() { _ = tRepos.Users.Delete(ctx, usr.ID) })

	require.NoError(t, tRepos.Roles.SetUserRoles(ctx, usr.ID, []uuid.UUID{viewEverywhere.ID, editOne.ID}))

	set, err := tRepos.Roles.GetUserPermissionSet(ctx, usr.ID)
	require.NoError(t, err)

	assert.True(t, set.Can(permissions.SectionItems, permissions.ActionView, uuid.New()), "all-collections view grant")
	assert.True(t, set.Can(permissions.SectionItems, permissions.ActionEdit, collection.ID), "union grants edit on scoped collection")
	assert.False(t, set.Can(permissions.SectionItems, permissions.ActionEdit, uuid.New()), "edit must not leak")
	assert.False(t, set.SuperAdmin)

	// Role deletion cascades grant rows and removes access.
	require.NoError(t, tRepos.Roles.Delete(ctx, editOne.ID))
	set, err = tRepos.Roles.GetUserPermissionSet(ctx, usr.ID)
	require.NoError(t, err)
	assert.False(t, set.Can(permissions.SectionItems, permissions.ActionEdit, collection.ID))
}

func TestRoleRepo_SuperAdminTracking(t *testing.T) {
	ctx := context.Background()

	superID, err := tRepos.Roles.EnsureSuperAdmin(ctx)
	require.NoError(t, err)

	// EnsureSuperAdmin is idempotent.
	again, err := tRepos.Roles.EnsureSuperAdmin(ctx)
	require.NoError(t, err)
	assert.Equal(t, superID, again)

	usr, err := tRepos.Users.Create(ctx, userFactory())
	require.NoError(t, err)
	t.Cleanup(func() { _ = tRepos.Users.Delete(ctx, usr.ID) })

	before, err := tRepos.Roles.CountSuperAdminUsers(ctx)
	require.NoError(t, err)

	require.NoError(t, tRepos.Roles.SetUserRoles(ctx, usr.ID, []uuid.UUID{superID}))

	has, err := tRepos.Roles.UserHasSuperAdmin(ctx, usr.ID)
	require.NoError(t, err)
	assert.True(t, has)

	set, err := tRepos.Roles.GetUserPermissionSet(ctx, usr.ID)
	require.NoError(t, err)
	assert.True(t, set.SuperAdmin)

	after, err := tRepos.Roles.CountSuperAdminUsers(ctx)
	require.NoError(t, err)
	assert.Equal(t, before+1, after)
}

func TestRoleRepo_CollectionCascadeDeletesGrants(t *testing.T) {
	ctx := context.Background()

	collection, err := tRepos.Groups.GroupCreate(ctx, "cascade-grants")
	require.NoError(t, err)

	role, err := tRepos.Roles.Create(ctx, roleFactory(RolePermissionInput{
		Section:      string(permissions.SectionItems),
		CollectionID: &collection.ID,
		CanView:      true,
	}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = tRepos.Roles.Delete(ctx, role.ID) })

	require.NoError(t, tRepos.Groups.GroupDelete(ctx, collection.ID))

	out, err := tRepos.Roles.GetOneID(ctx, role.ID)
	require.NoError(t, err)
	assert.Empty(t, out.Permissions, "grants must cascade away with the collection")
}
