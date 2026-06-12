package v1

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/hay-kot/httpkit/errchain"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/web/adapters"
	"go.opentelemetry.io/otel/attribute"
)

type EntityBulkDeleteRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1,max=500"`
}

// HandleEntitiesBulkEdit godoc
//
//	@Summary		Bulk Edit Entities
//	@Description	Moves, tags/untags, or archives many entities in one call
//	@Tags			Items
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		repo.EntityBulkEdit	true	"Bulk edit payload"
//	@Success		200		{object}	ActionAmountResult
//	@Router			/v1/entities/bulk [Post]
//	@Security		Bearer
func (ctrl *V1Controller) HandleEntitiesBulkEdit() errchain.HandlerFunc {
	fn := func(r *http.Request, body repo.EntityBulkEdit) (ActionAmountResult, error) {
		spanCtx, span := startEntityCtrlSpan(r.Context(), "controller.V1.HandleEntitiesBulkEdit",
			attribute.Int("entities.count", len(body.IDs)),
		)
		defer span.End()

		ctx := services.NewContext(spanCtx)
		completed, err := ctrl.repo.Entities.BulkEdit(ctx, ctx.GID, body)
		if err != nil {
			recordCtrlSpanError(span, err)
			return ActionAmountResult{}, err
		}
		return ActionAmountResult{Completed: completed}, nil
	}

	return adapters.Action(fn, http.StatusOK)
}

// HandleEntitiesBulkDelete godoc
//
//	@Summary		Bulk Delete Entities
//	@Description	Deletes many entities (and their attachments) in one call
//	@Tags			Items
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		EntityBulkDeleteRequest	true	"Bulk delete payload"
//	@Success		200		{object}	ActionAmountResult
//	@Router			/v1/entities/bulk/delete [Post]
//	@Security		Bearer
func (ctrl *V1Controller) HandleEntitiesBulkDelete() errchain.HandlerFunc {
	fn := func(r *http.Request, body EntityBulkDeleteRequest) (ActionAmountResult, error) {
		spanCtx, span := startEntityCtrlSpan(r.Context(), "controller.V1.HandleEntitiesBulkDelete",
			attribute.Int("entities.count", len(body.IDs)),
		)
		defer span.End()

		ctx := services.NewContext(spanCtx)
		completed, err := ctrl.repo.Entities.BulkDelete(ctx, ctx.GID, body.IDs)
		if err != nil {
			recordCtrlSpanError(span, err)
			return ActionAmountResult{Completed: completed}, err
		}
		return ActionAmountResult{Completed: completed}, nil
	}

	return adapters.Action(fn, http.StatusOK)
}
