package services

import (
	"errors"

	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
)

type GroupService struct {
	repos *repo.AllRepos
	// user service owns the default-data seeding helpers reused on
	// collection creation.
	users *UserService
}

func (svc *GroupService) UpdateGroup(ctx Context, data repo.GroupUpdate) (repo.Group, error) {
	if data.Name == "" {
		return repo.Group{}, errors.New("group name cannot be empty")
	}

	if data.Currency == "" {
		return repo.Group{}, errors.New("currency cannot be empty")
	}

	return svc.repos.Groups.GroupUpdate(ctx.Context, ctx.GID, data)
}

// CreateGroup creates a site-owned collection seeded with the default tags
// and locations. Access to it is controlled by role permissions.
func (svc *GroupService) CreateGroup(ctx Context, name string) (repo.Group, error) {
	if name == "" {
		return repo.Group{}, errors.New("group name cannot be empty")
	}

	group, err := svc.repos.Groups.GroupCreate(ctx.Context, name)
	if err != nil {
		return repo.Group{}, err
	}

	if err := svc.users.bootstrapCollectionDefaults(ctx.Context, group.ID); err != nil {
		return repo.Group{}, err
	}
	return group, nil
}

func (svc *GroupService) DeleteGroup(ctx Context) error {
	return svc.repos.Groups.GroupDelete(ctx.Context, ctx.GID)
}
