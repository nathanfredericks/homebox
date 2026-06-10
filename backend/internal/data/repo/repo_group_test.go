package repo

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/core/permissions"
)

func Test_Group_Create(t *testing.T) {
	g, err := tRepos.Groups.GroupCreate(context.Background(), "test")

	require.NoError(t, err)
	assert.Equal(t, "test", g.Name)

	// Get by ID
	foundGroup, err := tRepos.Groups.GroupByID(context.Background(), g.ID)
	require.NoError(t, err)
	assert.Equal(t, g.ID, foundGroup.ID)
}

func Test_Group_Update(t *testing.T) {
	g, err := tRepos.Groups.GroupCreate(context.Background(), "test")
	require.NoError(t, err)

	g, err = tRepos.Groups.GroupUpdate(context.Background(), g.ID, GroupUpdate{
		Name:     "test2",
		Currency: "eur",
	})
	require.NoError(t, err)
	assert.Equal(t, "test2", g.Name)
	assert.Equal(t, "EUR", g.Currency)
}

func Test_Group_GetAccessible(t *testing.T) {
	ctx := context.Background()

	visible, err := tRepos.Groups.GroupCreate(ctx, "accessible-visible")
	require.NoError(t, err)
	hidden, err := tRepos.Groups.GroupCreate(ctx, "accessible-hidden")
	require.NoError(t, err)

	containsGroup := func(groups []Group, id uuid.UUID) bool {
		for _, g := range groups {
			if g.ID == id {
				return true
			}
		}
		return false
	}

	// Super admins see everything.
	all, err := tRepos.Groups.GetAccessible(ctx, permissions.NewSet(true, nil))
	require.NoError(t, err)
	assert.True(t, containsGroup(all, visible.ID))
	assert.True(t, containsGroup(all, hidden.ID))

	// A per-collection view grant exposes only that collection.
	scoped, err := tRepos.Groups.GetAccessible(ctx, permissions.NewSet(false, []permissions.Grant{
		{Section: permissions.SectionItems, CollectionID: &visible.ID, Actions: permissions.ActionView},
	}))
	require.NoError(t, err)
	assert.True(t, containsGroup(scoped, visible.ID))
	assert.False(t, containsGroup(scoped, hidden.ID))

	// No grants: no collections exist for the user.
	none, err := tRepos.Groups.GetAccessible(ctx, permissions.NewSet(false, nil))
	require.NoError(t, err)
	assert.Empty(t, none)

	// An all-collections grant exposes everything.
	allScope, err := tRepos.Groups.GetAccessible(ctx, permissions.NewSet(false, []permissions.Grant{
		{Section: permissions.SectionItems, CollectionID: nil, Actions: permissions.ActionView},
	}))
	require.NoError(t, err)
	assert.True(t, containsGroup(allScope, visible.ID))
	assert.True(t, containsGroup(allScope, hidden.ID))
}
