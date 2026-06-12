package repo

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/attachment"
	"github.com/sysadminsmedia/homebox/backend/internal/data/types"
)

func containerFactory() EntityCreate {
	return EntityCreate{
		Name:        fk.Str(10),
		Description: fk.Str(100),
	}
}

func entityFactory() EntityCreate {
	return EntityCreate{
		Name:        fk.Str(10),
		Description: fk.Str(100),
	}
}

// useContainerEntityType creates or gets a default location entity type for the test group.
func useContainerEntityType(t *testing.T) EntityTypeSummary {
	t.Helper()
	et, err := tRepos.EntityTypes.GetDefault(context.Background(), tGroup.ID, true)
	require.NoError(t, err)
	return et
}

// useItemEntityType creates or gets a default item entity type for the test group.
func useItemEntityType(t *testing.T) EntityTypeSummary {
	t.Helper()
	et, err := tRepos.EntityTypes.GetDefault(context.Background(), tGroup.ID, false)
	require.NoError(t, err)
	return et
}

func useEntities(t *testing.T, count int) []EntityOut {
	t.Helper()

	containerET := useContainerEntityType(t)
	itemET := useItemEntityType(t)

	// Create a container entity
	cf := containerFactory()
	cf.EntityTypeID = containerET.ID
	container, err := tRepos.Entities.Create(context.Background(), tGroup.ID, cf)
	require.NoError(t, err)

	entities := make([]EntityOut, count)
	for i := 0; i < count; i++ {
		itm := entityFactory()
		itm.ParentID = container.ID
		itm.EntityTypeID = itemET.ID

		e, err := tRepos.Entities.Create(context.Background(), tGroup.ID, itm)
		require.NoError(t, err)
		entities[i] = e
	}

	t.Cleanup(func() {
		for _, e := range entities {
			_ = tRepos.Entities.Delete(context.Background(), e.ID)
		}
		_ = tRepos.Entities.Delete(context.Background(), container.ID)
	})

	return entities
}

func TestEntityRepository_RecursiveRelationships(t *testing.T) {
	parent := useEntities(t, 1)[0]

	children := useEntities(t, 3)

	for _, child := range children {
		update := EntityUpdate{
			ID:          child.ID,
			ParentID:    parent.ID,
			Name:        "note-important",
			Description: "This is a note",
		}
		if child.EntityType != nil {
			update.EntityTypeID = child.EntityType.ID
		}

		// Append Parent ID
		_, err := tRepos.Entities.UpdateByGroup(context.Background(), tGroup.ID, update)
		require.NoError(t, err)

		// Check Parent ID
		updated, err := tRepos.Entities.GetOne(context.Background(), child.ID)
		require.NoError(t, err)
		assert.Equal(t, parent.ID, updated.Parent.ID)

		// Remove Parent ID
		update.ParentID = uuid.Nil
		_, err = tRepos.Entities.UpdateByGroup(context.Background(), tGroup.ID, update)
		require.NoError(t, err)

		// Check Parent ID
		updated, err = tRepos.Entities.GetOne(context.Background(), child.ID)
		require.NoError(t, err)
		assert.Nil(t, updated.Parent)
	}
}

func TestEntityRepository_GetOne(t *testing.T) {
	entities := useEntities(t, 3)

	for _, e := range entities {
		result, err := tRepos.Entities.GetOne(context.Background(), e.ID)
		require.NoError(t, err)
		assert.Equal(t, e.ID, result.ID)
	}
}

func TestEntityRepository_GetAll(t *testing.T) {
	length := 10
	expected := useEntities(t, length)

	results, err := tRepos.Entities.GetAll(context.Background(), tGroup.ID)
	require.NoError(t, err)

	// Results include the container + the items
	assert.GreaterOrEqual(t, len(results), length)

	for _, e := range expected {
		found := false
		for _, r := range results {
			if e.ID == r.ID {
				found = true
				assert.Equal(t, e.Name, r.Name)
				assert.Equal(t, e.Description, r.Description)
			}
		}
		assert.True(t, found, "expected entity not found in results")
	}
}

func TestEntityRepository_Create(t *testing.T) {
	containerET := useContainerEntityType(t)
	itemET := useItemEntityType(t)

	cf := containerFactory()
	cf.EntityTypeID = containerET.ID
	container, err := tRepos.Entities.Create(context.Background(), tGroup.ID, cf)
	require.NoError(t, err)

	itm := entityFactory()
	itm.ParentID = container.ID
	itm.EntityTypeID = itemET.ID

	result, err := tRepos.Entities.Create(context.Background(), tGroup.ID, itm)
	require.NoError(t, err)
	assert.NotEmpty(t, result.ID)

	// Cleanup
	err = tRepos.Entities.Delete(context.Background(), result.ID)
	require.NoError(t, err)
	err = tRepos.Entities.Delete(context.Background(), container.ID)
	require.NoError(t, err)
}

func TestEntityRepository_Create_WithFractionalQuantity(t *testing.T) {
	containerET := useContainerEntityType(t)
	itemET := useItemEntityType(t)

	cf := containerFactory()
	cf.EntityTypeID = containerET.ID
	container, err := tRepos.Entities.Create(context.Background(), tGroup.ID, cf)
	require.NoError(t, err)

	itm := entityFactory()
	itm.ParentID = container.ID
	itm.EntityTypeID = itemET.ID
	itm.Quantity = 1.25

	result, err := tRepos.Entities.Create(context.Background(), tGroup.ID, itm)
	require.NoError(t, err)
	assert.NotEmpty(t, result.ID)
	assert.InDelta(t, 1.25, result.Quantity, 0.000001)

	fetched, err := tRepos.Entities.GetOne(context.Background(), result.ID)
	require.NoError(t, err)
	assert.InDelta(t, 1.25, fetched.Quantity, 0.000001)

	// Cleanup
	err = tRepos.Entities.Delete(context.Background(), result.ID)
	require.NoError(t, err)
	err = tRepos.Entities.Delete(context.Background(), container.ID)
	require.NoError(t, err)
}

func TestEntityRepository_Create_RejectsNonFiniteQuantity(t *testing.T) {
	containerET := useContainerEntityType(t)
	itemET := useItemEntityType(t)

	cf := containerFactory()
	cf.EntityTypeID = containerET.ID
	container, err := tRepos.Entities.Create(context.Background(), tGroup.ID, cf)
	require.NoError(t, err)

	itm := entityFactory()
	itm.ParentID = container.ID
	itm.EntityTypeID = itemET.ID
	itm.Quantity = math.NaN()

	_, err = tRepos.Entities.Create(context.Background(), tGroup.ID, itm)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quantity: must be a finite number")

	// Cleanup
	err = tRepos.Entities.Delete(context.Background(), container.ID)
	require.NoError(t, err)
}

func TestEntityRepository_Create_WithParent(t *testing.T) {
	containerET := useContainerEntityType(t)
	itemET := useItemEntityType(t)

	cf := containerFactory()
	cf.EntityTypeID = containerET.ID
	container, err := tRepos.Entities.Create(context.Background(), tGroup.ID, cf)
	require.NoError(t, err)
	assert.NotEmpty(t, container.ID)

	itm := entityFactory()
	itm.ParentID = container.ID
	itm.EntityTypeID = itemET.ID

	// Create Resource
	result, err := tRepos.Entities.Create(context.Background(), tGroup.ID, itm)
	require.NoError(t, err)
	assert.NotEmpty(t, result.ID)

	// Get Resource
	foundEntity, err := tRepos.Entities.GetOne(context.Background(), result.ID)
	require.NoError(t, err)
	assert.Equal(t, result.ID, foundEntity.ID)
	assert.NotNil(t, foundEntity.Parent)
	assert.Equal(t, container.ID, foundEntity.Parent.ID)

	// Cleanup
	err = tRepos.Entities.Delete(context.Background(), result.ID)
	require.NoError(t, err)
	err = tRepos.Entities.Delete(context.Background(), container.ID)
	require.NoError(t, err)
}

func TestEntityRepository_Delete(t *testing.T) {
	entities := useEntities(t, 3)

	for _, e := range entities {
		err := tRepos.Entities.Delete(context.Background(), e.ID)
		require.NoError(t, err)
	}

	results, err := tRepos.Entities.GetAll(context.Background(), tGroup.ID)
	require.NoError(t, err)
	// After deleting items, only container(s) remain
	for _, e := range entities {
		for _, r := range results {
			assert.NotEqual(t, e.ID, r.ID)
		}
	}
}

func TestEntityRepository_Update_Tags(t *testing.T) {
	e := useEntities(t, 1)[0]
	tags := useTags(t, 3)

	tagsIDs := []uuid.UUID{tags[0].ID, tags[1].ID, tags[2].ID}

	type args struct {
		tagIds []uuid.UUID
	}

	tests := []struct {
		name string
		args args
		want []uuid.UUID
	}{
		{
			name: "add all tags",
			args: args{
				tagIds: tagsIDs,
			},
			want: tagsIDs,
		},
		{
			name: "update with one tag",
			args: args{
				tagIds: tagsIDs[:1],
			},
			want: tagsIDs[:1],
		},
		{
			name: "add one new tag to existing single tag",
			args: args{
				tagIds: tagsIDs[1:],
			},
			want: tagsIDs[1:],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateData := EntityUpdate{
				ID:     e.ID,
				Name:   e.Name,
				TagIDs: tt.args.tagIds,
			}
			if e.EntityType != nil {
				updateData.EntityTypeID = e.EntityType.ID
			}

			updated, err := tRepos.Entities.UpdateByGroup(context.Background(), tGroup.ID, updateData)
			require.NoError(t, err)
			assert.Len(t, tt.want, len(updated.Tags))

			for _, tag := range updated.Tags {
				assert.Contains(t, tt.want, tag.ID)
			}
		})
	}
}

func TestEntityRepository_Update(t *testing.T) {
	entities := useEntities(t, 3)

	e := entities[0]

	updateData := EntityUpdate{
		ID:               e.ID,
		Name:             e.Name,
		SerialNumber:     fk.Str(10),
		TagIDs:           nil,
		ModelNumber:      fk.Str(10),
		Manufacturer:     fk.Str(10),
		PurchaseDate:     types.DateFromTime(time.Now()),
		PurchaseFrom:     fk.Str(10),
		PurchasePrice:    300.99,
		SoldDate:         types.DateFromTime(time.Now()),
		SoldTo:           fk.Str(10),
		SoldPrice:        300.99,
		SoldNotes:        fk.Str(10),
		Notes:            fk.Str(10),
		WarrantyExpires:  types.DateFromTime(time.Now()),
		WarrantyDetails:  fk.Str(10),
		LifetimeWarranty: true,
	}
	if e.EntityType != nil {
		updateData.EntityTypeID = e.EntityType.ID
	}

	updatedEntity, err := tRepos.Entities.UpdateByGroup(context.Background(), tGroup.ID, updateData)
	require.NoError(t, err)

	got, err := tRepos.Entities.GetOne(context.Background(), updatedEntity.ID)
	require.NoError(t, err)

	assert.Equal(t, updateData.ID, got.ID)
	assert.Equal(t, updateData.Name, got.Name)
	assert.Equal(t, updateData.SerialNumber, got.SerialNumber)
	assert.Equal(t, updateData.ModelNumber, got.ModelNumber)
	assert.Equal(t, updateData.Manufacturer, got.Manufacturer)
	assert.Equal(t, updateData.PurchaseFrom, got.PurchaseFrom)
	assert.InDelta(t, updateData.PurchasePrice, got.PurchasePrice, 0.01)
	assert.Equal(t, updateData.SoldTo, got.SoldTo)
	assert.InDelta(t, updateData.SoldPrice, got.SoldPrice, 0.01)
	assert.Equal(t, updateData.SoldNotes, got.SoldNotes)
	assert.Equal(t, updateData.Notes, got.Notes)
	assert.Equal(t, updateData.WarrantyDetails, got.WarrantyDetails)
	assert.Equal(t, updateData.LifetimeWarranty, got.LifetimeWarranty)
}

func TestEntityRepository_Update_WithFractionalQuantity(t *testing.T) {
	e := useEntities(t, 1)[0]

	updateData := EntityUpdate{
		ID:       e.ID,
		Name:     e.Name,
		Quantity: 2.75,
	}
	if e.EntityType != nil {
		updateData.EntityTypeID = e.EntityType.ID
	}

	updatedEntity, err := tRepos.Entities.UpdateByGroup(context.Background(), tGroup.ID, updateData)
	require.NoError(t, err)

	got, err := tRepos.Entities.GetOne(context.Background(), updatedEntity.ID)
	require.NoError(t, err)

	assert.InDelta(t, 2.75, got.Quantity, 0.000001)
}

func TestEntityRepository_Update_RejectsNonFiniteQuantity(t *testing.T) {
	e := useEntities(t, 1)[0]

	updateData := EntityUpdate{
		ID:       e.ID,
		Name:     e.Name,
		Quantity: math.Inf(1),
	}

	_, err := tRepos.Entities.UpdateByGroup(context.Background(), tGroup.ID, updateData)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quantity: must be a finite number")
}

func TestEntityRepository_Patch_RejectsNonFiniteQuantity(t *testing.T) {
	e := useEntities(t, 1)[0]

	quantity := math.Inf(-1)
	err := tRepos.Entities.Patch(context.Background(), tGroup.ID, e.ID, EntityPatch{Quantity: &quantity})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quantity: must be a finite number")
}

func TestEntityRepository_CreateFromTemplate_RejectsNonFiniteQuantity(t *testing.T) {
	containerET := useContainerEntityType(t)

	cf := containerFactory()
	cf.EntityTypeID = containerET.ID
	container, err := tRepos.Entities.Create(context.Background(), tGroup.ID, cf)
	require.NoError(t, err)

	_, err = tRepos.Entities.CreateFromTemplate(context.Background(), tGroup.ID, EntityCreateFromTemplate{
		Name:        fk.Str(10),
		Description: fk.Str(20),
		Quantity:    math.NaN(),
		ParentID:    container.ID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quantity: must be a finite number")

	// Cleanup
	err = tRepos.Entities.Delete(context.Background(), container.ID)
	require.NoError(t, err)
}

func TestEntityRepository_GetAllCustomFields(t *testing.T) {
	const FieldsCount = 5

	e := useEntities(t, 1)[0]

	fields := make([]EntityFieldData, FieldsCount)
	names := make([]string, FieldsCount)
	values := make([]string, FieldsCount)

	for i := 0; i < FieldsCount; i++ {
		name := fk.Str(10)
		fields[i] = EntityFieldData{
			Name:      name,
			Type:      "text",
			TextValue: fk.Str(10),
		}
		names[i] = name
		values[i] = fields[i].TextValue
	}

	updateData := EntityUpdate{
		ID:     e.ID,
		Name:   e.Name,
		Fields: fields,
	}
	if e.EntityType != nil {
		updateData.EntityTypeID = e.EntityType.ID
	}

	_, err := tRepos.Entities.UpdateByGroup(context.Background(), tGroup.ID, updateData)
	require.NoError(t, err)

	// Test getting all fields
	{
		results, err := tRepos.Entities.GetAllCustomFieldNames(context.Background(), tGroup.ID)
		require.NoError(t, err)
		assert.ElementsMatch(t, names, results)
	}

	// Test getting all values from field
	{
		results, err := tRepos.Entities.GetAllCustomFieldValues(context.Background(), tUser.DefaultGroupID, names[0])

		require.NoError(t, err)
		assert.ElementsMatch(t, values[:1], results)
	}
}

func TestEntityRepository_DeleteWithAttachments(t *testing.T) {
	// Create an entity with an attachment
	e := useEntities(t, 1)[0]

	// Add an attachment to the entity
	att, err := tRepos.Attachments.Create(
		context.Background(),
		e.ID,
		ItemCreateAttachment{
			Title:   "test-attachment.txt",
			Content: strings.NewReader("test content for attachment deletion"),
		},
		attachment.TypePhoto,
		true,
	)
	require.NoError(t, err)
	assert.NotNil(t, att)

	// Verify the attachment exists
	retrievedAttachment, err := tRepos.Attachments.Get(context.Background(), tGroup.ID, att.ID)
	require.NoError(t, err)
	assert.Equal(t, att.ID, retrievedAttachment.ID)

	// Verify the attachment is linked to the entity
	entityWithAttachments, err := tRepos.Entities.GetOne(context.Background(), e.ID)
	require.NoError(t, err)
	assert.Len(t, entityWithAttachments.Attachments, 1)
	assert.Equal(t, att.ID, entityWithAttachments.Attachments[0].ID)

	// Delete the entity
	err = tRepos.Entities.Delete(context.Background(), e.ID)
	require.NoError(t, err)

	// Verify the entity is deleted
	_, err = tRepos.Entities.GetOne(context.Background(), e.ID)
	require.Error(t, err)

	// Verify the attachment is also deleted
	_, err = tRepos.Attachments.Get(context.Background(), tGroup.ID, att.ID)
	require.Error(t, err)
}

func TestEntityRepository_DeleteByGroupWithAttachments(t *testing.T) {
	// Create an entity with an attachment
	e := useEntities(t, 1)[0]

	// Add an attachment to the entity
	att, err := tRepos.Attachments.Create(
		context.Background(),
		e.ID,
		ItemCreateAttachment{
			Title:   "test-attachment-by-group.txt",
			Content: strings.NewReader("test content for attachment deletion by group"),
		},
		attachment.TypePhoto,
		true,
	)
	require.NoError(t, err)
	assert.NotNil(t, att)

	// Verify the attachment exists
	retrievedAttachment, err := tRepos.Attachments.Get(context.Background(), tGroup.ID, att.ID)
	require.NoError(t, err)
	assert.Equal(t, att.ID, retrievedAttachment.ID)

	// Delete the entity using DeleteByGroup
	err = tRepos.Entities.DeleteByGroup(context.Background(), tGroup.ID, e.ID)
	require.NoError(t, err)

	// Verify the entity is deleted
	_, err = tRepos.Entities.GetOneByGroup(context.Background(), tGroup.ID, e.ID)
	require.Error(t, err)

	// Verify the attachment is also deleted
	_, err = tRepos.Attachments.Get(context.Background(), tGroup.ID, att.ID)
	require.Error(t, err)
}

func strptr(s string) *string   { return &s }
func f64ptr(f float64) *float64 { return &f }
func boolptr(b bool) *bool      { return &b }

func TestEntityRepository_Patch_ScalarFields(t *testing.T) {
	e := useEntities(t, 1)[0]
	ctx := context.Background()

	err := tRepos.Entities.Patch(ctx, tGroup.ID, e.ID, EntityPatch{
		ID:            e.ID,
		Name:          strptr("Patched Name"),
		Description:   strptr("patched description"),
		SerialNumber:  strptr("SN-PATCH"),
		ModelNumber:   strptr("MODEL-1"),
		Manufacturer:  strptr("ACME"),
		Notes:         strptr("patched notes"),
		PurchaseFrom:  strptr("Hardware Store"),
		PurchasePrice: f64ptr(42.5),
	})
	require.NoError(t, err)

	got, err := tRepos.Entities.GetOneByGroup(ctx, tGroup.ID, e.ID)
	require.NoError(t, err)
	assert.Equal(t, "Patched Name", got.Name)
	assert.Equal(t, "patched description", got.Description)
	assert.Equal(t, "SN-PATCH", got.SerialNumber)
	assert.Equal(t, "MODEL-1", got.ModelNumber)
	assert.Equal(t, "ACME", got.Manufacturer)
	assert.Equal(t, "patched notes", got.Notes)
	assert.Equal(t, "Hardware Store", got.PurchaseFrom)
	assert.InDelta(t, 42.5, got.PurchasePrice, 0.001)

	// Nil fields must leave existing values untouched.
	err = tRepos.Entities.Patch(ctx, tGroup.ID, e.ID, EntityPatch{
		ID:    e.ID,
		Notes: strptr(""),
	})
	require.NoError(t, err)

	got, err = tRepos.Entities.GetOneByGroup(ctx, tGroup.ID, e.ID)
	require.NoError(t, err)
	assert.Equal(t, "Patched Name", got.Name, "untouched field must keep its value")
	assert.Empty(t, got.Notes, "pointed-to empty string must clear the field")
}

func TestEntityRepository_GetBySerialNumber(t *testing.T) {
	e := useEntities(t, 1)[0]
	ctx := context.Background()

	require.NoError(t, tRepos.Entities.Patch(ctx, tGroup.ID, e.ID, EntityPatch{
		ID:           e.ID,
		SerialNumber: strptr("Dup-Serial-1"),
	}))

	matches, err := tRepos.Entities.GetBySerialNumber(ctx, tGroup.ID, "dup-serial-1")
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, e.ID, matches[0].ID)

	// Archived entities are excluded.
	require.NoError(t, tRepos.Entities.Patch(ctx, tGroup.ID, e.ID, EntityPatch{
		ID:       e.ID,
		Archived: boolptr(true),
	}))
	matches, err = tRepos.Entities.GetBySerialNumber(ctx, tGroup.ID, "dup-serial-1")
	require.NoError(t, err)
	assert.Empty(t, matches)
}

func TestEntityRepository_BulkEdit(t *testing.T) {
	entities := useEntities(t, 3)
	tags := useTags(t, 2)
	ctx := context.Background()

	containerET := useContainerEntityType(t)
	cf := containerFactory()
	cf.EntityTypeID = containerET.ID
	newParent, err := tRepos.Entities.Create(ctx, tGroup.ID, cf)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tRepos.Entities.Delete(ctx, newParent.ID) })

	ids := []uuid.UUID{entities[0].ID, entities[1].ID}
	completed, err := tRepos.Entities.BulkEdit(ctx, tGroup.ID, EntityBulkEdit{
		IDs:       ids,
		ParentID:  newParent.ID,
		AddTagIDs: []uuid.UUID{tags[0].ID, tags[1].ID},
		Archived:  boolptr(true),
	})
	require.NoError(t, err)
	assert.Equal(t, 2, completed)

	for _, id := range ids {
		got, err := tRepos.Entities.GetOneByGroup(ctx, tGroup.ID, id)
		require.NoError(t, err)
		require.NotNil(t, got.Parent)
		assert.Equal(t, newParent.ID, got.Parent.ID)
		assert.True(t, got.Archived)
		assert.Len(t, got.Tags, 2)
	}

	// Third entity untouched.
	got, err := tRepos.Entities.GetOneByGroup(ctx, tGroup.ID, entities[2].ID)
	require.NoError(t, err)
	assert.False(t, got.Archived)
	assert.Empty(t, got.Tags)

	// Remove one tag and unarchive.
	_, err = tRepos.Entities.BulkEdit(ctx, tGroup.ID, EntityBulkEdit{
		IDs:          ids,
		RemoveTagIDs: []uuid.UUID{tags[0].ID},
		Archived:     boolptr(false),
	})
	require.NoError(t, err)
	got, err = tRepos.Entities.GetOneByGroup(ctx, tGroup.ID, ids[0])
	require.NoError(t, err)
	assert.False(t, got.Archived)
	require.Len(t, got.Tags, 1)
	assert.Equal(t, tags[1].ID, got.Tags[0].ID)
}

func TestEntityRepository_BulkEdit_RejectsCrossGroup(t *testing.T) {
	entities := useEntities(t, 1)
	ctx := context.Background()

	otherGroup, err := tRepos.Groups.GroupCreate(ctx, "bulk-other-group")
	require.NoError(t, err)
	otherET, err := tRepos.EntityTypes.GetDefault(ctx, otherGroup.ID, false)
	require.NoError(t, err)
	foreign, err := tRepos.Entities.Create(ctx, otherGroup.ID, EntityCreate{Name: "foreign", EntityTypeID: otherET.ID})
	require.NoError(t, err)
	t.Cleanup(func() { _ = tRepos.Entities.Delete(ctx, foreign.ID) })

	_, err = tRepos.Entities.BulkEdit(ctx, tGroup.ID, EntityBulkEdit{
		IDs:      []uuid.UUID{entities[0].ID, foreign.ID},
		Archived: boolptr(true),
	})
	require.Error(t, err, "batch containing a foreign entity must fail")

	// Nothing applied to the in-group entity.
	got, err := tRepos.Entities.GetOneByGroup(ctx, tGroup.ID, entities[0].ID)
	require.NoError(t, err)
	assert.False(t, got.Archived)
}

func TestEntityRepository_BulkDelete(t *testing.T) {
	entities := useEntities(t, 2)
	ctx := context.Background()

	completed, err := tRepos.Entities.BulkDelete(ctx, tGroup.ID, []uuid.UUID{entities[0].ID, entities[1].ID})
	require.NoError(t, err)
	assert.Equal(t, 2, completed)

	for _, e := range entities {
		_, err := tRepos.Entities.GetOneByGroup(ctx, tGroup.ID, e.ID)
		require.Error(t, err)
	}
}
