-- +goose Up
-- Seed the built-in "Admin" and "User" groups so new instances ship with
-- sensible defaults: Admin grants every section, User grants day-to-day
-- inventory access. Both are ordinary editable roles (unlike Super Admin);
-- the guards keep this from resurrecting them if an admin renames or
-- deletes them later, and skip seeding when the names are already taken.

INSERT INTO roles (id, created_at, updated_at, name, description, is_super_admin)
SELECT '00000000-0000-0000-0000-000000000002', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'Admin', 'Full administrative access to every section and collection.', false
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000002' OR name = 'Admin');

INSERT INTO roles (id, created_at, updated_at, name, description, is_super_admin)
SELECT '00000000-0000-0000-0000-000000000003', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'User', 'Standard inventory access: manage items, locations, tags, templates, maintenance and notifiers.', false
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000003' OR name = 'User');

-- Admin: every section, every action, all collections.
INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0002-000000000001', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'items', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000002'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000002')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0002-000000000001');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0002-000000000002', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'locations', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000002'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000002')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0002-000000000002');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0002-000000000003', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'tags', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000002'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000002')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0002-000000000003');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0002-000000000004', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'templates', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000002'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000002')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0002-000000000004');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0002-000000000005', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'maintenance', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000002'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000002')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0002-000000000005');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0002-000000000006', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'statistics', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000002'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000002')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0002-000000000006');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0002-000000000007', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'collection_settings', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000002'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000002')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0002-000000000007');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0002-000000000008', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'entity_types', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000002'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000002')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0002-000000000008');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0002-000000000009', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'notifiers', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000002'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000002')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0002-000000000009');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0002-000000000010', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'tools', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000002'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000002')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0002-000000000010');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0002-000000000011', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'users', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000002'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000002')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0002-000000000011');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0002-000000000012', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'roles', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000002'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000002')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0002-000000000012');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0002-000000000013', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'collections', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000002'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000002')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0002-000000000013');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0002-000000000014', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'site_settings', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000002'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000002')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0002-000000000014');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0002-000000000015', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'theming', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000002'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000002')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0002-000000000015');

-- User: full inventory access, read-only supporting sections, no admin sections.
INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0003-000000000001', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'items', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000003'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000003')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0003-000000000001');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0003-000000000002', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'locations', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000003'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000003')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0003-000000000002');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0003-000000000003', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'tags', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000003'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000003')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0003-000000000003');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0003-000000000004', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'templates', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000003'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000003')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0003-000000000004');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0003-000000000005', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'maintenance', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000003'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000003')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0003-000000000005');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0003-000000000006', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'statistics', true, false, false, false, NULL, '00000000-0000-0000-0000-000000000003'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000003')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0003-000000000006');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0003-000000000007', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'entity_types', true, false, false, false, NULL, '00000000-0000-0000-0000-000000000003'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000003')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0003-000000000007');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0003-000000000008', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'notifiers', true, true, true, true, NULL, '00000000-0000-0000-0000-000000000003'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000003')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0003-000000000008');

INSERT INTO role_permissions (id, created_at, updated_at, section, can_view, can_create, can_edit, can_delete, collection_id, role_id)
SELECT '00000000-0000-0000-0003-000000000009', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'tools', true, false, false, false, NULL, '00000000-0000-0000-0000-000000000003'
WHERE EXISTS (SELECT 1 FROM roles WHERE id = '00000000-0000-0000-0000-000000000003')
  AND NOT EXISTS (SELECT 1 FROM role_permissions WHERE id = '00000000-0000-0000-0003-000000000009');
