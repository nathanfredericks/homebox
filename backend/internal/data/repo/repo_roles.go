package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/sysadminsmedia/homebox/backend/internal/core/permissions"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/role"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/rolepermission"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/user"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// SuperAdminRoleID is the fixed ID of the seeded Super Admin role. The role
// is created by migration and re-ensured at startup; it cannot be edited or
// deleted.
var SuperAdminRoleID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type RoleRepository struct {
	db *ent.Client
}

type (
	// RolePermissionInput is one matrix row when creating/updating a role.
	// CollectionID nil = all collections (or site scope for site sections).
	RolePermissionInput struct {
		Section      string     `json:"section"`
		CollectionID *uuid.UUID `json:"collectionId" extensions:"x-nullable"`
		CanView      bool       `json:"canView"`
		CanCreate    bool       `json:"canCreate"`
		CanEdit      bool       `json:"canEdit"`
		CanDelete    bool       `json:"canDelete"`
	}

	RoleCreate struct {
		Name        string                `json:"name" validate:"required,max=255"`
		Description string                `json:"description" validate:"max=1000"`
		Permissions []RolePermissionInput `json:"permissions"`
	}

	RoleUpdate = RoleCreate

	RolePermissionOut struct {
		Section      string     `json:"section"`
		CollectionID *uuid.UUID `json:"collectionId" extensions:"x-nullable"`
		CanView      bool       `json:"canView"`
		CanCreate    bool       `json:"canCreate"`
		CanEdit      bool       `json:"canEdit"`
		CanDelete    bool       `json:"canDelete"`
	}

	RoleSummary struct {
		ID           uuid.UUID `json:"id"`
		Name         string    `json:"name"`
		IsSuperAdmin bool      `json:"isSuperAdmin"`
	}

	RoleOut struct {
		ID           uuid.UUID           `json:"id"`
		Name         string              `json:"name"`
		Description  string              `json:"description"`
		IsSuperAdmin bool                `json:"isSuperAdmin"`
		UserCount    int                 `json:"userCount"`
		Permissions  []RolePermissionOut `json:"permissions"`
	}
)

func mapRoleSummary(r *ent.Role) RoleSummary {
	return RoleSummary{
		ID:           r.ID,
		Name:         r.Name,
		IsSuperAdmin: r.IsSuperAdmin,
	}
}

func mapRolePermissionOut(p *ent.RolePermission) RolePermissionOut {
	return RolePermissionOut{
		Section:      p.Section,
		CollectionID: p.CollectionID,
		CanView:      p.CanView,
		CanCreate:    p.CanCreate,
		CanEdit:      p.CanEdit,
		CanDelete:    p.CanDelete,
	}
}

func mapRoleOut(r *ent.Role) RoleOut {
	return RoleOut{
		ID:           r.ID,
		Name:         r.Name,
		Description:  r.Description,
		IsSuperAdmin: r.IsSuperAdmin,
		UserCount:    len(r.Edges.Users),
		Permissions: lo.Map(r.Edges.Permissions, func(p *ent.RolePermission, _ int) RolePermissionOut {
			return mapRolePermissionOut(p)
		}),
	}
}

func actionsOf(canView, canCreate, canEdit, canDelete bool) permissions.Action {
	var a permissions.Action
	if canView {
		a |= permissions.ActionView
	}
	if canCreate {
		a |= permissions.ActionCreate
	}
	if canEdit {
		a |= permissions.ActionEdit
	}
	if canDelete {
		a |= permissions.ActionDelete
	}
	return a
}

func (r *RoleRepository) GetAll(ctx context.Context) ([]RoleOut, error) {
	ctx, span := entityTracer().Start(ctx, "repo.RoleRepository.GetAll")
	defer span.End()

	roles, err := r.db.Role.Query().
		WithPermissions().
		WithUsers().
		Order(ent.Asc(role.FieldName)).
		All(ctx)
	if err != nil {
		recordSpanError(span, err)
		return nil, err
	}
	span.SetAttributes(attribute.Int("roles.count", len(roles)))
	return lo.Map(roles, func(r *ent.Role, _ int) RoleOut { return mapRoleOut(r) }), nil
}

func (r *RoleRepository) GetOneID(ctx context.Context, id uuid.UUID) (RoleOut, error) {
	ctx, span := entityTracer().Start(ctx, "repo.RoleRepository.GetOneID",
		trace.WithAttributes(attribute.String("role.id", id.String())))
	defer span.End()

	entRole, err := r.db.Role.Query().
		Where(role.ID(id)).
		WithPermissions().
		WithUsers().
		Only(ctx)
	if err != nil {
		recordSpanError(span, err)
		return RoleOut{}, err
	}
	return mapRoleOut(entRole), nil
}

func (r *RoleRepository) createPermissionRows(ctx context.Context, tx *ent.Tx, roleID uuid.UUID, perms []RolePermissionInput) error {
	for _, p := range perms {
		q := tx.RolePermission.Create().
			SetRoleID(roleID).
			SetSection(p.Section).
			SetCanView(p.CanView).
			SetCanCreate(p.CanCreate).
			SetCanEdit(p.CanEdit).
			SetCanDelete(p.CanDelete)
		if p.CollectionID != nil {
			q = q.SetCollectionID(*p.CollectionID)
		}
		if _, err := q.Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *RoleRepository) Create(ctx context.Context, data RoleCreate) (RoleOut, error) {
	ctx, span := entityTracer().Start(ctx, "repo.RoleRepository.Create")
	defer span.End()

	tx, err := r.db.Tx(ctx)
	if err != nil {
		recordSpanError(span, err)
		return RoleOut{}, err
	}

	entRole, err := tx.Role.Create().
		SetName(data.Name).
		SetDescription(data.Description).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		recordSpanError(span, err)
		return RoleOut{}, err
	}

	if err := r.createPermissionRows(ctx, tx, entRole.ID, data.Permissions); err != nil {
		_ = tx.Rollback()
		recordSpanError(span, err)
		return RoleOut{}, err
	}

	if err := tx.Commit(); err != nil {
		recordSpanError(span, err)
		return RoleOut{}, err
	}
	span.SetAttributes(attribute.String("role.id", entRole.ID.String()))
	return r.GetOneID(ctx, entRole.ID)
}

// Update replaces the role's name, description and full permission set in a
// single transaction (delete-all + insert keeps the unique index honest for
// NULL collection scopes).
func (r *RoleRepository) Update(ctx context.Context, id uuid.UUID, data RoleUpdate) (RoleOut, error) {
	ctx, span := entityTracer().Start(ctx, "repo.RoleRepository.Update",
		trace.WithAttributes(attribute.String("role.id", id.String())))
	defer span.End()

	tx, err := r.db.Tx(ctx)
	if err != nil {
		recordSpanError(span, err)
		return RoleOut{}, err
	}

	if err := tx.Role.UpdateOneID(id).
		SetName(data.Name).
		SetDescription(data.Description).
		Exec(ctx); err != nil {
		_ = tx.Rollback()
		recordSpanError(span, err)
		return RoleOut{}, err
	}

	if _, err := tx.RolePermission.Delete().
		Where(rolepermission.RoleID(id)).
		Exec(ctx); err != nil {
		_ = tx.Rollback()
		recordSpanError(span, err)
		return RoleOut{}, err
	}

	if err := r.createPermissionRows(ctx, tx, id, data.Permissions); err != nil {
		_ = tx.Rollback()
		recordSpanError(span, err)
		return RoleOut{}, err
	}

	if err := tx.Commit(); err != nil {
		recordSpanError(span, err)
		return RoleOut{}, err
	}
	return r.GetOneID(ctx, id)
}

func (r *RoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, span := entityTracer().Start(ctx, "repo.RoleRepository.Delete",
		trace.WithAttributes(attribute.String("role.id", id.String())))
	defer span.End()

	err := r.db.Role.DeleteOneID(id).Exec(ctx)
	recordSpanError(span, err)
	return err
}

// EnsureSuperAdmin idempotently creates the seeded Super Admin role. Called
// at startup as a safety net alongside the migration seed.
func (r *RoleRepository) EnsureSuperAdmin(ctx context.Context) (uuid.UUID, error) {
	ctx, span := entityTracer().Start(ctx, "repo.RoleRepository.EnsureSuperAdmin")
	defer span.End()

	existing, err := r.db.Role.Query().Where(role.IsSuperAdmin(true)).First(ctx)
	if err == nil {
		return existing.ID, nil
	}
	if !ent.IsNotFound(err) {
		recordSpanError(span, err)
		return uuid.Nil, err
	}

	created, err := r.db.Role.Create().
		SetID(SuperAdminRoleID).
		SetName("Super Admin").
		SetDescription("Full access to everything. This group cannot be edited or deleted.").
		SetIsSuperAdmin(true).
		Save(ctx)
	if err != nil {
		recordSpanError(span, err)
		return uuid.Nil, err
	}
	return created.ID, nil
}

// GetUserPermissionSet assembles the user's effective permissions: the union
// of all grant rows across their roles, with Super Admin short-circuiting.
func (r *RoleRepository) GetUserPermissionSet(ctx context.Context, userID uuid.UUID) (*permissions.Set, error) {
	ctx, span := entityTracer().Start(ctx, "repo.RoleRepository.GetUserPermissionSet",
		trace.WithAttributes(attribute.String("user.id", userID.String())))
	defer span.End()

	roles, err := r.db.Role.Query().
		Where(role.HasUsersWith(user.ID(userID))).
		WithPermissions().
		All(ctx)
	if err != nil {
		recordSpanError(span, err)
		return nil, err
	}

	superAdmin := false
	var grants []permissions.Grant
	for _, entRole := range roles {
		if entRole.IsSuperAdmin {
			superAdmin = true
			continue
		}
		for _, p := range entRole.Edges.Permissions {
			grants = append(grants, permissions.Grant{
				Section:      permissions.Section(p.Section),
				CollectionID: p.CollectionID,
				Actions:      actionsOf(p.CanView, p.CanCreate, p.CanEdit, p.CanDelete),
			})
		}
	}
	span.SetAttributes(
		attribute.Int("roles.count", len(roles)),
		attribute.Bool("user.is_super_admin", superAdmin),
	)
	return permissions.NewSet(superAdmin, grants), nil
}

func (r *RoleRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]RoleSummary, error) {
	ctx, span := entityTracer().Start(ctx, "repo.RoleRepository.GetUserRoles",
		trace.WithAttributes(attribute.String("user.id", userID.String())))
	defer span.End()

	roles, err := r.db.Role.Query().
		Where(role.HasUsersWith(user.ID(userID))).
		Order(ent.Asc(role.FieldName)).
		All(ctx)
	if err != nil {
		recordSpanError(span, err)
		return nil, err
	}
	return lo.Map(roles, func(r *ent.Role, _ int) RoleSummary { return mapRoleSummary(r) }), nil
}

// SetUserRoles replaces the user's role assignments.
func (r *RoleRepository) SetUserRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	ctx, span := entityTracer().Start(ctx, "repo.RoleRepository.SetUserRoles",
		trace.WithAttributes(
			attribute.String("user.id", userID.String()),
			attribute.Int("roles.count", len(roleIDs)),
		))
	defer span.End()

	err := r.db.User.UpdateOneID(userID).
		ClearRoles().
		AddRoleIDs(roleIDs...).
		Exec(ctx)
	recordSpanError(span, err)
	return err
}

// UserHasSuperAdmin reports whether the user holds a super-admin role.
func (r *RoleRepository) UserHasSuperAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	ctx, span := entityTracer().Start(ctx, "repo.RoleRepository.UserHasSuperAdmin",
		trace.WithAttributes(attribute.String("user.id", userID.String())))
	defer span.End()

	exists, err := r.db.Role.Query().
		Where(role.IsSuperAdmin(true), role.HasUsersWith(user.ID(userID))).
		Exist(ctx)
	recordSpanError(span, err)
	return exists, err
}

// CountSuperAdminUsers returns how many users hold a super-admin role. The
// system must never drop below one.
func (r *RoleRepository) CountSuperAdminUsers(ctx context.Context) (int, error) {
	ctx, span := entityTracer().Start(ctx, "repo.RoleRepository.CountSuperAdminUsers")
	defer span.End()

	n, err := r.db.User.Query().
		Where(user.HasRolesWith(role.IsSuperAdmin(true))).
		Count(ctx)
	if err != nil {
		recordSpanError(span, err)
		return 0, err
	}
	span.SetAttributes(attribute.Int("users.super_admin.count", n))
	return n, nil
}
