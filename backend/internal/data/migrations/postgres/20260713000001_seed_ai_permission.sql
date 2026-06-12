-- +goose Up
-- The "ai" permission section landed after the default groups were seeded, so
-- existing installs have Admin/User roles without it and AI features stay
-- invisible to every non-superadmin even when enabled. Grant view (the only
-- action the section exposes) to both seeded roles, guarded the same way as
-- the original seed so deleted roles are not resurrected and reruns are no-ops.

INSERT INTO "role_permissions" ("id", "created_at", "updated_at", "section", "can_view", "can_create", "can_edit", "can_delete", "collection_id", "role_id")
SELECT '00000000-0000-0000-0002-000000000016', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'ai', true, false, false, false, NULL, '00000000-0000-0000-0000-000000000002'
WHERE EXISTS (SELECT 1 FROM "roles" WHERE "id" = '00000000-0000-0000-0000-000000000002')
  AND NOT EXISTS (SELECT 1 FROM "role_permissions" WHERE "id" = '00000000-0000-0000-0002-000000000016')
  AND NOT EXISTS (SELECT 1 FROM "role_permissions" WHERE "role_id" = '00000000-0000-0000-0000-000000000002' AND "section" = 'ai');

INSERT INTO "role_permissions" ("id", "created_at", "updated_at", "section", "can_view", "can_create", "can_edit", "can_delete", "collection_id", "role_id")
SELECT '00000000-0000-0000-0003-000000000010', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'ai', true, false, false, false, NULL, '00000000-0000-0000-0000-000000000003'
WHERE EXISTS (SELECT 1 FROM "roles" WHERE "id" = '00000000-0000-0000-0000-000000000003')
  AND NOT EXISTS (SELECT 1 FROM "role_permissions" WHERE "id" = '00000000-0000-0000-0003-000000000010')
  AND NOT EXISTS (SELECT 1 FROM "role_permissions" WHERE "role_id" = '00000000-0000-0000-0000-000000000003' AND "section" = 'ai');
