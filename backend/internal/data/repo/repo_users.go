package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/user"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type UserRepository struct {
	db *ent.Client
}

type (
	// UserCreate is the data object containing the requirements of creating a
	// user in the database. Users hold no inherent access; permissions come
	// entirely from assigned roles.
	UserCreate struct {
		Name           string    `json:"name"`
		Email          string    `json:"email"`
		Password       *string   `json:"password"`
		DefaultGroupID uuid.UUID `json:"defaultGroupID"`
	}

	UserUpdate struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	UserOut struct {
		ID             uuid.UUID `json:"id"`
		Name           string    `json:"name"`
		Email          string    `json:"email"`
		DefaultGroupID uuid.UUID `json:"defaultGroupId"`
		PasswordHash   string    `json:"-"`
		OidcIssuer     *string   `json:"oidcIssuer"`
		OidcSubject    *string   `json:"oidcSubject"`
	}

	UserSummary struct {
		Name  string    `json:"name"`
		Email string    `json:"email"`
		ID    uuid.UUID `json:"id"`
	}

	// UserAdminOut is the admin-facing user shape including assigned roles.
	UserAdminOut struct {
		UserOut
		Roles []RoleSummary `json:"roles"`
	}
)

var (
	mapUserOutErr = mapTErrFunc(mapUserOut)
)

func mapUserOut(user *ent.User) UserOut {
	return UserOut{
		ID:             user.ID,
		Name:           user.Name,
		Email:          user.Email,
		DefaultGroupID: lo.FromPtrOr(user.DefaultGroupID, uuid.Nil),
		PasswordHash:   lo.FromPtrOr(user.Password, ""),
		OidcIssuer:     user.OidcIssuer,
		OidcSubject:    user.OidcSubject,
	}
}

func mapUserAdminOut(usr *ent.User) UserAdminOut {
	return UserAdminOut{
		UserOut: mapUserOut(usr),
		Roles: lo.Map(usr.Edges.Roles, func(r *ent.Role, _ int) RoleSummary {
			return mapRoleSummary(r)
		}),
	}
}

func userSpanAttrs(out UserOut) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("user.id", out.ID.String()),
		attribute.String("user.default_group_id", out.DefaultGroupID.String()),
		attribute.Bool("user.has_password_hash", out.PasswordHash != ""),
		attribute.Bool("user.has_oidc", out.OidcIssuer != nil && out.OidcSubject != nil),
	}
}

func (r *UserRepository) GetOneID(ctx context.Context, id uuid.UUID) (UserOut, error) {
	ctx, span := entityTracer().Start(ctx, "repo.UserRepository.GetOneID",
		trace.WithAttributes(attribute.String("user.id", id.String())))
	defer span.End()

	out, err := mapUserOutErr(r.db.User.Query().
		Where(user.ID(id)).
		Only(ctx))
	if err != nil {
		recordSpanError(span, err)
		return out, err
	}
	span.SetAttributes(userSpanAttrs(out)...)
	return out, nil
}

func (r *UserRepository) GetOneEmail(ctx context.Context, email string) (UserOut, error) {
	ctx, span := entityTracer().Start(ctx, "repo.UserRepository.GetOneEmail",
		trace.WithAttributes(attribute.Int("user.email.length", len(email))))
	defer span.End()

	out, err := mapUserOutErr(r.db.User.Query().
		Where(user.EmailEqualFold(email)).
		Only(ctx),
	)
	if err != nil {
		// "not found" is expected on bad logins; record on the span but don't mark
		// it as an error status so dashboards aren't flooded with red.
		span.SetAttributes(
			attribute.Bool("user.found", false),
			attribute.String("user.lookup.error", err.Error()),
			attribute.Bool("user.lookup.not_found", ent.IsNotFound(err)),
		)
		if !ent.IsNotFound(err) {
			recordSpanError(span, err)
		}
		return out, err
	}
	span.SetAttributes(attribute.Bool("user.found", true))
	span.SetAttributes(userSpanAttrs(out)...)
	return out, nil
}

func (r *UserRepository) GetOneEmailNoEdges(ctx context.Context, email string) (UserOut, error) {
	ctx, span := entityTracer().Start(ctx, "repo.UserRepository.GetOneEmailNoEdges",
		trace.WithAttributes(attribute.Int("user.email.length", len(email))))
	defer span.End()

	out, err := mapUserOutErr(r.db.User.Query().
		Where(user.EmailEqualFold(email)).
		Only(ctx),
	)
	if err != nil {
		span.SetAttributes(
			attribute.Bool("user.found", false),
			attribute.Bool("user.lookup.not_found", ent.IsNotFound(err)),
		)
		if !ent.IsNotFound(err) {
			recordSpanError(span, err)
		}
		return out, err
	}
	span.SetAttributes(attribute.Bool("user.found", true))
	span.SetAttributes(userSpanAttrs(out)...)
	return out, nil
}

func (r *UserRepository) GetAll(ctx context.Context) ([]UserAdminOut, error) {
	ctx, span := entityTracer().Start(ctx, "repo.UserRepository.GetAll")
	defer span.End()

	users, err := r.db.User.Query().
		WithRoles().
		Order(ent.Asc(user.FieldName)).
		All(ctx)
	if err != nil {
		recordSpanError(span, err)
		return nil, err
	}
	out := lo.Map(users, func(u *ent.User, _ int) UserAdminOut {
		return mapUserAdminOut(u)
	})
	span.SetAttributes(attribute.Int("users.count", len(out)))
	return out, nil
}

// Count returns the total number of users. Zero means the instance is
// awaiting first-time setup.
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	ctx, span := entityTracer().Start(ctx, "repo.UserRepository.Count")
	defer span.End()

	n, err := r.db.User.Query().Count(ctx)
	if err != nil {
		recordSpanError(span, err)
		return 0, err
	}
	span.SetAttributes(attribute.Int("users.count", n))
	return n, nil
}

func (r *UserRepository) create(
	ctx context.Context,
	usr UserCreate,
	configure func(*ent.UserCreate) *ent.UserCreate,
) (uuid.UUID, error) {
	q := r.db.User.
		Create().
		SetName(usr.Name).
		SetEmail(usr.Email)

	// Admin-created users have no default collection until they pick one;
	// a zero UUID would violate the FK, so only set non-nil values.
	if usr.DefaultGroupID != uuid.Nil {
		q = q.SetDefaultGroupID(usr.DefaultGroupID)
	}
	if usr.Password != nil {
		q = q.SetPassword(*usr.Password)
	}
	if configure != nil {
		q = configure(q)
	}

	entUser, err := q.Save(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	return entUser.ID, nil
}

func (r *UserRepository) Create(ctx context.Context, usr UserCreate) (UserOut, error) {
	ctx, span := entityTracer().Start(ctx, "repo.UserRepository.Create",
		trace.WithAttributes(
			attribute.String("user.default_group_id", usr.DefaultGroupID.String()),
			attribute.Bool("user.has_password", usr.Password != nil),
		))
	defer span.End()

	id, err := r.create(ctx, usr, nil)
	if err != nil {
		recordSpanError(span, err)
		return UserOut{}, err
	}
	span.SetAttributes(attribute.String("user.id", id.String()))

	out, err := r.GetOneID(ctx, id)
	if err != nil {
		recordSpanError(span, err)
	}
	return out, err
}

func (r *UserRepository) CreateWithOIDC(ctx context.Context, usr UserCreate, issuer, subject string) (UserOut, error) {
	ctx, span := entityTracer().Start(ctx, "repo.UserRepository.CreateWithOIDC",
		trace.WithAttributes(
			attribute.String("user.default_group_id", usr.DefaultGroupID.String()),
			attribute.Bool("user.has_password", usr.Password != nil),
			attribute.String("oidc.issuer", issuer),
			attribute.Int("oidc.subject.length", len(subject)),
		))
	defer span.End()

	id, err := r.create(ctx, usr, func(uc *ent.UserCreate) *ent.UserCreate {
		return uc.SetOidcIssuer(issuer).SetOidcSubject(subject)
	})
	if err != nil {
		recordSpanError(span, err)
		return UserOut{}, err
	}
	span.SetAttributes(attribute.String("user.id", id.String()))

	out, err := r.GetOneID(ctx, id)
	if err != nil {
		recordSpanError(span, err)
	}
	return out, err
}

func (r *UserRepository) Update(ctx context.Context, id uuid.UUID, data UserUpdate) error {
	ctx, span := entityTracer().Start(ctx, "repo.UserRepository.Update",
		trace.WithAttributes(attribute.String("user.id", id.String())))
	defer span.End()

	q := r.db.User.Update().
		Where(user.ID(id)).
		SetName(data.Name).
		SetEmail(data.Email)

	_, err := q.Save(ctx)
	recordSpanError(span, err)
	return err
}

func (r *UserRepository) UpdateDefaultGroup(ctx context.Context, id uuid.UUID, groupID uuid.UUID) error {
	ctx, span := entityTracer().Start(ctx, "repo.UserRepository.UpdateDefaultGroup",
		trace.WithAttributes(
			attribute.String("user.id", id.String()),
			attribute.String("group.id", groupID.String()),
		))
	defer span.End()

	err := r.db.User.UpdateOneID(id).SetDefaultGroupID(groupID).Exec(ctx)
	recordSpanError(span, err)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, span := entityTracer().Start(ctx, "repo.UserRepository.Delete",
		trace.WithAttributes(attribute.String("user.id", id.String())))
	defer span.End()

	_, err := r.db.User.Delete().Where(user.ID(id)).Exec(ctx)
	recordSpanError(span, err)
	return err
}

func (r *UserRepository) DeleteAll(ctx context.Context) error {
	ctx, span := entityTracer().Start(ctx, "repo.UserRepository.DeleteAll")
	defer span.End()

	_, err := r.db.User.Delete().Exec(ctx)
	recordSpanError(span, err)
	return err
}

func (r *UserRepository) ChangePassword(ctx context.Context, uid uuid.UUID, pw string) error {
	ctx, span := entityTracer().Start(ctx, "repo.UserRepository.ChangePassword",
		trace.WithAttributes(
			attribute.String("user.id", uid.String()),
			attribute.Int("password.hash.length", len(pw)),
		))
	defer span.End()

	err := r.db.User.UpdateOneID(uid).SetPassword(pw).Exec(ctx)
	recordSpanError(span, err)
	return err
}

func (r *UserRepository) SetSettings(ctx context.Context, uid uuid.UUID, settings map[string]interface{}) error {
	ctx, span := entityTracer().Start(ctx, "repo.UserRepository.SetSettings",
		trace.WithAttributes(
			attribute.String("user.id", uid.String()),
			attribute.Int("settings.keys.count", len(settings)),
		))
	defer span.End()

	err := r.db.User.UpdateOneID(uid).SetSettings(settings).Exec(ctx)
	recordSpanError(span, err)
	return err
}

func (r *UserRepository) GetSettings(ctx context.Context, uid uuid.UUID) (map[string]interface{}, error) {
	ctx, span := entityTracer().Start(ctx, "repo.UserRepository.GetSettings",
		trace.WithAttributes(attribute.String("user.id", uid.String())))
	defer span.End()

	usr, err := r.db.User.Get(ctx, uid)
	if err != nil {
		recordSpanError(span, err)
		return nil, err
	}
	span.SetAttributes(attribute.Int("settings.keys.count", len(usr.Settings)))
	return usr.Settings, nil
}

func (r *UserRepository) SetOIDCIdentity(ctx context.Context, uid uuid.UUID, issuer, subject string) error {
	ctx, span := entityTracer().Start(ctx, "repo.UserRepository.SetOIDCIdentity",
		trace.WithAttributes(
			attribute.String("user.id", uid.String()),
			attribute.String("oidc.issuer", issuer),
			attribute.Int("oidc.subject.length", len(subject)),
		))
	defer span.End()

	err := r.db.User.UpdateOneID(uid).SetOidcIssuer(issuer).SetOidcSubject(subject).Exec(ctx)
	recordSpanError(span, err)
	return err
}

func (r *UserRepository) GetOneOIDC(ctx context.Context, issuer, subject string) (UserOut, error) {
	ctx, span := entityTracer().Start(ctx, "repo.UserRepository.GetOneOIDC",
		trace.WithAttributes(
			attribute.String("oidc.issuer", issuer),
			attribute.Int("oidc.subject.length", len(subject)),
		))
	defer span.End()

	out, err := mapUserOutErr(r.db.User.Query().
		Where(user.OidcIssuerEQ(issuer), user.OidcSubjectEQ(subject)).
		Only(ctx))
	if err != nil {
		span.SetAttributes(
			attribute.Bool("user.found", false),
			attribute.Bool("user.lookup.not_found", ent.IsNotFound(err)),
		)
		if !ent.IsNotFound(err) {
			recordSpanError(span, err)
		}
		return out, err
	}
	span.SetAttributes(attribute.Bool("user.found", true))
	span.SetAttributes(userSpanAttrs(out)...)
	return out, nil
}
