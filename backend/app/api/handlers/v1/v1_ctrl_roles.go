package v1

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/hay-kot/httpkit/errchain"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
	"github.com/sysadminsmedia/homebox/backend/internal/web/adapters"
)

// mapRoleServiceError converts role service sentinel errors into request
// errors with the right status codes.
func mapRoleServiceError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, services.ErrSuperAdminImmutable),
		errors.Is(err, services.ErrLastSuperAdmin):
		return validate.NewRequestError(err, http.StatusConflict)
	case errors.Is(err, services.ErrInvalidPermission):
		return validate.NewRequestError(err, http.StatusUnprocessableEntity)
	default:
		return err
	}
}

// HandleRolesGetAll godoc
//
//	@Summary	Get All Roles (Groups)
//	@Tags		Roles
//	@Produce	json
//	@Success	200	{object}	[]repo.RoleOut
//	@Router		/v1/roles [Get]
//	@Security	Bearer
func (ctrl *V1Controller) HandleRolesGetAll() errchain.HandlerFunc {
	fn := func(r *http.Request) ([]repo.RoleOut, error) {
		return ctrl.svc.Roles.GetAll(r.Context())
	}

	return adapters.Command(fn, http.StatusOK)
}

// HandleRoleGet godoc
//
//	@Summary	Get Role (Group)
//	@Tags		Roles
//	@Produce	json
//	@Param		id	path		string	true	"Role ID"
//	@Success	200	{object}	repo.RoleOut
//	@Router		/v1/roles/{id} [Get]
//	@Security	Bearer
func (ctrl *V1Controller) HandleRoleGet() errchain.HandlerFunc {
	fn := func(r *http.Request, id uuid.UUID) (repo.RoleOut, error) {
		return ctrl.svc.Roles.GetOne(r.Context(), id)
	}

	return adapters.CommandID("id", fn, http.StatusOK)
}

// HandleRoleCreate godoc
//
//	@Summary	Create Role (Group)
//	@Tags		Roles
//	@Produce	json
//	@Param		payload	body		repo.RoleCreate	true	"Role Data"
//	@Success	201		{object}	repo.RoleOut
//	@Router		/v1/roles [Post]
//	@Security	Bearer
func (ctrl *V1Controller) HandleRoleCreate() errchain.HandlerFunc {
	fn := func(r *http.Request, body repo.RoleCreate) (repo.RoleOut, error) {
		out, err := ctrl.svc.Roles.Create(r.Context(), body)
		return out, mapRoleServiceError(err)
	}

	return adapters.Action(fn, http.StatusCreated)
}

// HandleRoleUpdate godoc
//
//	@Summary	Update Role (Group)
//	@Tags		Roles
//	@Produce	json
//	@Param		id		path		string			true	"Role ID"
//	@Param		payload	body		repo.RoleUpdate	true	"Role Data"
//	@Success	200		{object}	repo.RoleOut
//	@Router		/v1/roles/{id} [Put]
//	@Security	Bearer
func (ctrl *V1Controller) HandleRoleUpdate() errchain.HandlerFunc {
	fn := func(r *http.Request, id uuid.UUID, body repo.RoleUpdate) (repo.RoleOut, error) {
		out, err := ctrl.svc.Roles.Update(r.Context(), id, body)
		return out, mapRoleServiceError(err)
	}

	return adapters.ActionID("id", fn, http.StatusOK)
}

// HandleRoleDelete godoc
//
//	@Summary	Delete Role (Group)
//	@Tags		Roles
//	@Produce	json
//	@Param		id	path	string	true	"Role ID"
//	@Success	204
//	@Router		/v1/roles/{id} [Delete]
//	@Security	Bearer
func (ctrl *V1Controller) HandleRoleDelete() errchain.HandlerFunc {
	fn := func(r *http.Request, id uuid.UUID) (any, error) {
		err := ctrl.svc.Roles.Delete(r.Context(), id)
		return nil, mapRoleServiceError(err)
	}

	return adapters.CommandID("id", fn, http.StatusNoContent)
}
