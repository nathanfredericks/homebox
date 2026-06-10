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

func mapUserAdminServiceError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, services.ErrLastSuperAdmin):
		return validate.NewRequestError(err, http.StatusConflict)
	case errors.Is(err, services.ErrorPasswordTooShort):
		return validate.NewRequestError(err, http.StatusUnprocessableEntity)
	default:
		return err
	}
}

// HandleAdminUsersGetAll godoc
//
//	@Summary	Get All Users
//	@Tags		Users
//	@Produce	json
//	@Success	200	{object}	[]repo.UserAdminOut
//	@Router		/v1/users [Get]
//	@Security	Bearer
func (ctrl *V1Controller) HandleAdminUsersGetAll() errchain.HandlerFunc {
	fn := func(r *http.Request) ([]repo.UserAdminOut, error) {
		return ctrl.svc.User.AdminList(r.Context())
	}

	return adapters.Command(fn, http.StatusOK)
}

// HandleAdminUserCreate godoc
//
//	@Summary	Create User
//	@Tags		Users
//	@Produce	json
//	@Param		payload	body		services.UserAdminCreate	true	"User Data"
//	@Success	201		{object}	repo.UserAdminOut
//	@Router		/v1/users [Post]
//	@Security	Bearer
func (ctrl *V1Controller) HandleAdminUserCreate() errchain.HandlerFunc {
	fn := func(r *http.Request, body services.UserAdminCreate) (repo.UserAdminOut, error) {
		out, err := ctrl.svc.User.AdminCreate(r.Context(), body)
		return out, mapUserAdminServiceError(err)
	}

	return adapters.Action(fn, http.StatusCreated)
}

// HandleAdminUserUpdate godoc
//
//	@Summary	Update User (name, email, roles, optional password reset)
//	@Tags		Users
//	@Produce	json
//	@Param		id		path		string					true	"User ID"
//	@Param		payload	body		services.UserAdminUpdate	true	"User Data"
//	@Success	200		{object}	repo.UserAdminOut
//	@Router		/v1/users/{id} [Put]
//	@Security	Bearer
func (ctrl *V1Controller) HandleAdminUserUpdate() errchain.HandlerFunc {
	fn := func(r *http.Request, id uuid.UUID, body services.UserAdminUpdate) (repo.UserAdminOut, error) {
		out, err := ctrl.svc.User.AdminUpdate(r.Context(), id, body)
		return out, mapUserAdminServiceError(err)
	}

	return adapters.ActionID("id", fn, http.StatusOK)
}

// HandleAdminUserDelete godoc
//
//	@Summary	Delete User
//	@Tags		Users
//	@Produce	json
//	@Param		id	path	string	true	"User ID"
//	@Success	204
//	@Router		/v1/users/{id} [Delete]
//	@Security	Bearer
func (ctrl *V1Controller) HandleAdminUserDelete() errchain.HandlerFunc {
	fn := func(r *http.Request, id uuid.UUID) (any, error) {
		err := ctrl.svc.User.AdminDelete(r.Context(), id)
		return nil, mapUserAdminServiceError(err)
	}

	return adapters.CommandID("id", fn, http.StatusNoContent)
}
