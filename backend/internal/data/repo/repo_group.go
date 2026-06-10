package repo

import (
	"context"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/sysadminsmedia/homebox/backend/internal/core/permissions"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/entity"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/entitytype"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/group"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/notifier"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/tag"
)

type GroupRepository struct {
	db          *ent.Client
	groupMapper MapFunc[*ent.Group, Group]
	attachments *AttachmentRepo
}

func NewGroupRepository(db *ent.Client, attachments *AttachmentRepo) *GroupRepository {
	gmap := func(g *ent.Group) Group {
		return Group{
			ID:        g.ID,
			Name:      g.Name,
			CreatedAt: g.CreatedAt,
			UpdatedAt: g.UpdatedAt,
			Currency:  strings.ToUpper(g.Currency),
		}
	}

	return &GroupRepository{
		db:          db,
		groupMapper: gmap,
		attachments: attachments,
	}
}

type (
	Group struct {
		ID        uuid.UUID `json:"id,omitempty"`
		Name      string    `json:"name,omitempty"`
		CreatedAt time.Time `json:"createdAt,omitempty"`
		UpdatedAt time.Time `json:"updatedAt,omitempty"`
		Currency  string    `json:"currency,omitempty"`
	}

	GroupUpdate struct {
		Name     string `json:"name"`
		Currency string `json:"currency"`
	}

	GroupStatistics struct {
		TotalUsers        int     `json:"totalUsers"`
		TotalItems        int     `json:"totalItems"`
		TotalLocations    int     `json:"totalLocations"`
		TotalTags         int     `json:"totalTags"`
		TotalItemPrice    float64 `json:"totalItemPrice"`
		TotalWithWarranty int     `json:"totalWithWarranty"`
	}

	ValueOverTimeEntry struct {
		Date  time.Time `json:"date"`
		Value float64   `json:"value"`
		Name  string    `json:"name"`
	}

	ValueOverTime struct {
		PriceAtStart float64              `json:"valueAtStart"`
		PriceAtEnd   float64              `json:"valueAtEnd"`
		Start        time.Time            `json:"start"`
		End          time.Time            `json:"end"`
		Entries      []ValueOverTimeEntry `json:"entries"`
	}

	TotalsByOrganizer struct {
		ID    uuid.UUID `json:"id"`
		Name  string    `json:"name"`
		Total float64   `json:"total"`
	}
)

// GetAllGroups returns every collection. For system/background use only —
// request paths must use GetAccessible.
func (r *GroupRepository) GetAllGroups(ctx context.Context) ([]Group, error) {
	return r.groupMapper.MapEachErr(r.db.Group.Query().All(ctx))
}

// GetAccessible returns the collections visible to a permission set: all of
// them for super admins, holders of collections:view or an all-collections
// grant, otherwise only the explicitly granted ones.
func (r *GroupRepository) GetAccessible(ctx context.Context, set *permissions.Set) ([]Group, error) {
	if set.Can(permissions.SectionCollections, permissions.ActionView, uuid.Nil) {
		return r.groupMapper.MapEachErr(r.db.Group.Query().All(ctx))
	}

	all, ids := set.AccessibleCollections()
	if all {
		return r.groupMapper.MapEachErr(r.db.Group.Query().All(ctx))
	}
	if len(ids) == 0 {
		return []Group{}, nil
	}
	return r.groupMapper.MapEachErr(r.db.Group.Query().Where(group.IDIn(ids...)).All(ctx))
}

func (r *GroupRepository) StatsLocationsByPurchasePrice(ctx context.Context, gid uuid.UUID) ([]TotalsByOrganizer, error) {
	var v []TotalsByOrganizer

	// Query entities that are containers (is_location=true) and sum purchase prices of their children
	q := `
		SELECT parent.id, parent.name,
			COALESCE(SUM(child.purchase_price), 0) AS total
		FROM entities parent
		JOIN entity_types et ON et.id = parent.entity_type_entities
		LEFT JOIN entities child ON child.entity_children = parent.id
			AND child.entity_type_entities IN (SELECT id FROM entity_types WHERE is_location = false)
		WHERE parent.group_entities = $1 AND et.is_location = true
		GROUP BY parent.id, parent.name
		HAVING COALESCE(SUM(child.purchase_price), 0) > 0
	`

	rows, err := r.db.Sql().QueryContext(ctx, q, gid)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var item TotalsByOrganizer
		if err := rows.Scan(&item.ID, &item.Name, &item.Total); err != nil {
			return nil, err
		}
		v = append(v, item)
	}

	return v, rows.Err()
}

func (r *GroupRepository) StatsTagsByPurchasePrice(ctx context.Context, gid uuid.UUID) ([]TotalsByOrganizer, error) {
	var v []TotalsByOrganizer

	err := r.db.Tag.Query().
		Where(
			tag.HasGroupWith(group.ID(gid)),
		).
		GroupBy(tag.FieldID, tag.FieldName).
		Aggregate(func(sq *sql.Selector) string {
			entityTable := sql.Table(entity.Table)

			jt := sql.Table(tag.EntitiesTable)

			sq.Join(jt).On(sq.C(tag.FieldID), jt.C(tag.EntitiesPrimaryKey[0]))
			sq.Join(entityTable).On(jt.C(tag.EntitiesPrimaryKey[1]), entityTable.C(entity.FieldID))

			return sql.As(sql.Sum(entityTable.C(entity.FieldPurchasePrice)), "total")
		}).
		Scan(ctx, &v)
	if err != nil {
		return nil, err
	}

	return v, err
}

func (r *GroupRepository) StatsPurchasePrice(ctx context.Context, gid uuid.UUID, start, end time.Time) (*ValueOverTime, error) {
	// Get the Totals for the Start and End of the Given Time Period
	q := `
	SELECT
		SUM(CASE WHEN e.created_at < $1 THEN e.purchase_price ELSE 0 END) AS price_at_start,
		SUM(CASE WHEN e.created_at < $2 THEN e.purchase_price ELSE 0 END) AS price_at_end
	FROM entities e
	JOIN entity_types et ON et.id = e.entity_type_entities
	WHERE e.group_entities = $3 AND e.archived = false AND et.is_location = false
`
	stats := ValueOverTime{
		Start: start,
		End:   end,
	}

	var maybeStart *float64
	var maybeEnd *float64

	row := r.db.Sql().QueryRowContext(ctx, q, sqliteDateFormat(start), sqliteDateFormat(end), gid)
	err := row.Scan(&maybeStart, &maybeEnd)
	if err != nil {
		return nil, err
	}

	stats.PriceAtStart = orDefault(maybeStart, 0)
	stats.PriceAtEnd = orDefault(maybeEnd, 0)

	type itemPriceEntry struct {
		Name          string    `json:"name"`
		CreatedAt     time.Time `json:"created_at"`
		PurchasePrice float64   `json:"purchase_price"`
	}

	var v []itemPriceEntry

	// Get Created Date and Price of all entities between start and end
	err = r.db.Entity.Query().
		Where(
			entity.HasGroupWith(group.ID(gid)),
			entity.CreatedAtGTE(start),
			entity.CreatedAtLTE(end),
			entity.Archived(false),
			entity.HasEntityTypeWith(entitytype.IsLocation(false)),
		).
		Select(
			entity.FieldName,
			entity.FieldCreatedAt,
			entity.FieldPurchasePrice,
		).
		Scan(ctx, &v)

	if err != nil {
		return nil, err
	}

	stats.Entries = lo.Map(v, func(vv itemPriceEntry, _ int) ValueOverTimeEntry {
		return ValueOverTimeEntry{
			Date:  vv.CreatedAt,
			Value: vv.PurchasePrice,
		}
	})

	return &stats, nil
}

func (r *GroupRepository) StatsGroup(ctx context.Context, gid uuid.UUID) (GroupStatistics, error) {
	// total_users counts users whose roles give them access to this
	// collection: super admins, or any view grant scoped to it (or to all
	// collections) on a collection-scoped section.
	q := `
		SELECT
            (SELECT COUNT(DISTINCT ur.user_id)
                FROM user_roles ur
                JOIN roles r ON r.id = ur.role_id
                LEFT JOIN role_permissions rp ON rp.role_id = r.id
                WHERE r.is_super_admin = true
                   OR (rp.can_view = true
                       AND (rp.collection_id IS NULL OR rp.collection_id = $2)
                       AND rp.section NOT IN ('users', 'roles', 'collections'))
            ) AS total_users,
            (SELECT COUNT(*) FROM entities e JOIN entity_types et ON et.id = e.entity_type_entities WHERE e.group_entities = $2 AND e.archived = false AND et.is_location = false) AS total_items,
            (SELECT COUNT(*) FROM entities e JOIN entity_types et ON et.id = e.entity_type_entities WHERE e.group_entities = $2 AND et.is_location = true) AS total_locations,
            (SELECT COUNT(*) FROM tags WHERE group_tags = $2) AS total_tags,
            (SELECT SUM(e.purchase_price*e.quantity) FROM entities e JOIN entity_types et ON et.id = e.entity_type_entities WHERE e.group_entities = $2 AND e.archived = false AND et.is_location = false) AS total_item_price,
            (SELECT COUNT(*)
                FROM entities e
                JOIN entity_types et ON et.id = e.entity_type_entities
                    WHERE e.group_entities = $2
                    AND e.archived = false
                    AND et.is_location = false
                    AND (e.lifetime_warranty = true OR e.warranty_expires > $1)
                ) AS total_with_warranty;
`
	var stats GroupStatistics
	row := r.db.Sql().QueryRowContext(ctx, q, sqliteDateFormat(time.Now()), gid)

	var maybeTotalItemPrice *float64
	var maybeTotalWithWarranty *int

	err := row.Scan(&stats.TotalUsers, &stats.TotalItems, &stats.TotalLocations, &stats.TotalTags, &maybeTotalItemPrice, &maybeTotalWithWarranty)
	if err != nil {
		return GroupStatistics{}, err
	}

	stats.TotalItemPrice = orDefault(maybeTotalItemPrice, 0)
	stats.TotalWithWarranty = orDefault(maybeTotalWithWarranty, 0)

	return stats, nil
}

// GroupCreate creates a collection. Collections belong to the site; access is
// granted through role permissions, so no membership rows are created.
func (r *GroupRepository) GroupCreate(ctx context.Context, name string) (Group, error) {
	return r.groupMapper.MapErr(r.db.Group.Create().SetName(name).Save(ctx))
}

func (r *GroupRepository) GroupUpdate(ctx context.Context, id uuid.UUID, data GroupUpdate) (Group, error) {
	entity, err := r.db.Group.UpdateOneID(id).
		SetName(data.Name).
		SetCurrency(strings.ToLower(data.Currency)).
		Save(ctx)

	return r.groupMapper.MapErr(entity, err)
}

func (r *GroupRepository) GroupByID(ctx context.Context, id uuid.UUID) (Group, error) {
	return r.groupMapper.MapErr(r.db.Group.Get(ctx, id))
}

func (r *GroupRepository) GroupDelete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return err
	}

	itm, err := tx.Entity.Query().
		Where(entity.HasGroupWith(group.ID(id))).
		WithGroup().
		WithAttachments().
		All(ctx)
	if err != nil {
		return err
	}

	// Delete all attachments (and their files) before deleting the entities
	for _, it := range itm {
		for _, att := range it.Edges.Attachments {
			if err := r.attachments.Delete(ctx, id, att.ID); err != nil {
				log.Err(err).Str("attachment_id", att.ID.String()).Msg("failed to delete attachment during entity deletion")
				// Continue with other attachments even if one fails
			}
		}
	}

	// Delete all entities from the database
	if _, err := tx.Entity.Delete().
		Where(entity.HasGroupWith(group.ID(id))).
		Exec(ctx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			log.Error().Err(rerr).Msg("failed to rollback transaction")
		}
		return err
	}

	// Delete any associated notifiers
	if _, err := tx.Notifier.Delete().
		Where(notifier.HasGroupWith(group.ID(id))).
		Exec(ctx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			log.Error().Err(rerr).Msg("failed to rollback transaction")
		}
		return err
	}

	// Delete the group
	if err := tx.Group.DeleteOneID(id).Exec(ctx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			log.Error().Err(rerr).Msg("failed to rollback transaction")
		}
		return err
	}

	return tx.Commit()
}
