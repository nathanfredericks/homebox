package v1

import (
	"net/http"

	"github.com/hay-kot/httpkit/errchain"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
	"github.com/sysadminsmedia/homebox/backend/internal/web/adapters"
)

type (
	CreateRequest struct {
		Name string `json:"name" validate:"required"`
	}
)

// HandleGroupGet godoc
//
//	@Summary	Get Group
//	@Tags		Group
//	@Produce	json
//	@Success	200	{object}	repo.Group
//	@Router		/v1/groups [Get]
//	@Security	Bearer
func (ctrl *V1Controller) HandleGroupGet() errchain.HandlerFunc {
	fn := func(r *http.Request) (repo.Group, error) {
		auth := services.NewContext(r.Context())
		return ctrl.repo.Groups.GroupByID(auth, auth.GID)
	}

	return adapters.Command(fn, http.StatusOK)
}

// HandleGroupUpdate godoc
//
//	@Summary	Update Group
//	@Tags		Group
//	@Produce	json
//	@Param		payload	body		repo.GroupUpdate	true	"User Data"
//	@Success	200		{object}	repo.Group
//	@Router		/v1/groups [Put]
//	@Security	Bearer
func (ctrl *V1Controller) HandleGroupUpdate() errchain.HandlerFunc {
	fn := func(r *http.Request, body repo.GroupUpdate) (repo.Group, error) {
		auth := services.NewContext(r.Context())

		ok := ctrl.svc.Currencies.IsSupported(body.Currency)
		if !ok {
			return repo.Group{}, validate.NewFieldErrors(
				validate.NewFieldError("currency", "currency '"+body.Currency+"' is not supported"),
			)
		}

		return ctrl.svc.Group.UpdateGroup(auth, body)
	}

	return adapters.Action(fn, http.StatusOK)
}

// HandleGroupsGetAll godoc
//
//	@Summary	Get All Accessible Groups
//	@Tags		Group
//	@Produce	json
//	@Success	200	{object}	[]repo.Group
//	@Router		/v1/groups/all [Get]
//	@Security	Bearer
func (ctrl *V1Controller) HandleGroupsGetAll() errchain.HandlerFunc {
	fn := func(r *http.Request) ([]repo.Group, error) {
		auth := services.NewContext(r.Context())
		// Collections the user cannot access are omitted entirely — to them
		// those collections do not exist.
		return ctrl.repo.Groups.GetAccessible(auth, auth.Perms)
	}

	return adapters.Command(fn, http.StatusOK)
}

// HandleGroupCreate godoc
//
//	@Summary	Create Group
//	@Tags		Group
//	@Produce	json
//	@Param		payload	body		CreateRequest	true	"Create group request"
//	@Success	201		{object}	repo.Group
//	@Router		/v1/groups [Post]
//	@Security	Bearer
func (ctrl *V1Controller) HandleGroupCreate() errchain.HandlerFunc {
	fn := func(r *http.Request, body CreateRequest) (repo.Group, error) {
		auth := services.NewContext(r.Context())
		return ctrl.svc.Group.CreateGroup(auth, body.Name)
	}

	return adapters.Action(fn, http.StatusCreated)
}

// HandleGroupDelete godoc
//
//	@Summary	Delete Group
//	@Tags		Group
//	@Produce	json
//	@Success	204
//	@Router		/v1/groups [Delete]
//	@Security	Bearer
func (ctrl *V1Controller) HandleGroupDelete() errchain.HandlerFunc {
	fn := func(r *http.Request) (any, error) {
		auth := services.NewContext(r.Context())

		// default_group_id references are cleared by the FK's ON DELETE SET
		// NULL; affected users fall back to their first accessible collection
		// on the next request (see mwTenant).
		err := ctrl.svc.Group.DeleteGroup(auth)
		return nil, err
	}

	return adapters.Command(fn, http.StatusNoContent)
}
