-- +goose Up
-- RBAC overhaul: collections belong to the site, not to users. Access is
-- granted through roles (shown as "Groups" in the UI) that carry a granular
-- per-collection permission matrix. Group membership (user_groups) and the
-- invitation system are removed. Existing group owners and legacy superusers
-- are backfilled into the seeded Super Admin role; all other users start with
-- no roles and must be granted access by an admin.

CREATE TABLE IF NOT EXISTS "roles" (
    "id"             uuid NOT NULL,
    "created_at"     timestamptz NOT NULL,
    "updated_at"     timestamptz NOT NULL,
    "name"           character varying NOT NULL,
    "description"    character varying NULL,
    "is_super_admin" boolean NOT NULL DEFAULT false,
    PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX IF NOT EXISTS "roles_name_key" ON "roles" ("name");

-- collection_id NULL = all collections (or the site scope for site-level sections).
CREATE TABLE IF NOT EXISTS "role_permissions" (
    "id"            uuid NOT NULL,
    "created_at"    timestamptz NOT NULL,
    "updated_at"    timestamptz NOT NULL,
    "section"       character varying NOT NULL,
    "can_view"      boolean NOT NULL DEFAULT false,
    "can_create"    boolean NOT NULL DEFAULT false,
    "can_edit"      boolean NOT NULL DEFAULT false,
    "can_delete"    boolean NOT NULL DEFAULT false,
    "collection_id" uuid NULL,
    "role_id"       uuid NOT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "role_permissions_groups_role_permissions" FOREIGN KEY ("collection_id") REFERENCES "groups" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "role_permissions_roles_permissions" FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS "rolepermission_role_id_section_collection_id" ON "role_permissions" ("role_id", "section", "collection_id");

CREATE TABLE IF NOT EXISTS "user_roles" (
    "user_id" uuid NOT NULL,
    "role_id" uuid NOT NULL,
    PRIMARY KEY ("user_id", "role_id"),
    CONSTRAINT "user_roles_user_id" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "user_roles_role_id" FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- Seed the Super Admin role (fixed UUID; evaluation short-circuits on
-- is_super_admin so it needs no permission rows).
INSERT INTO "roles" ("id", "created_at", "updated_at", "name", "description", "is_super_admin")
SELECT '00000000-0000-0000-0000-000000000001', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
       'Super Admin', 'Full access to everything. This group cannot be edited or deleted.', true
WHERE NOT EXISTS (SELECT 1 FROM "roles" WHERE "is_super_admin" = true);

-- Backfill: group owners and legacy superusers become Super Admins.
INSERT INTO "user_roles" ("user_id", "role_id")
SELECT DISTINCT u."id", '00000000-0000-0000-0000-000000000001'::uuid
FROM "users" u
WHERE u."is_superuser" = true
   OR u."superuser" = true
   OR EXISTS (SELECT 1 FROM "user_groups" ug WHERE ug."user_id" = u."id" AND ug."role" = 'owner');

DROP TABLE IF EXISTS "user_groups";
DROP TABLE IF EXISTS "group_invitation_tokens";

ALTER TABLE "users" DROP COLUMN IF EXISTS "is_superuser";
ALTER TABLE "users" DROP COLUMN IF EXISTS "superuser";

-- +goose Down
-- Best effort: membership data cannot be fully reconstructed. Users are
-- re-attached to their default group; Super Admins become its owner.
ALTER TABLE "users" ADD COLUMN IF NOT EXISTS "is_superuser" boolean NOT NULL DEFAULT false;
ALTER TABLE "users" ADD COLUMN IF NOT EXISTS "superuser" boolean NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS "user_groups" (
    "user_id"  uuid NOT NULL,
    "group_id" uuid NOT NULL,
    "role"     character varying NOT NULL DEFAULT 'user',
    PRIMARY KEY ("user_id", "group_id"),
    CONSTRAINT "user_groups_user_id" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "user_groups_group_id" FOREIGN KEY ("group_id") REFERENCES "groups" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

INSERT INTO "user_groups" ("user_id", "group_id", "role")
SELECT u."id",
       u."default_group_id",
       CASE
           WHEN EXISTS (SELECT 1
                        FROM "user_roles" ur
                        JOIN "roles" r ON r."id" = ur."role_id"
                        WHERE ur."user_id" = u."id"
                          AND r."is_super_admin" = true) THEN 'owner'
           ELSE 'user'
       END
FROM "users" u
WHERE u."default_group_id" IS NOT NULL;

CREATE TABLE IF NOT EXISTS "group_invitation_tokens" (
    "id"         uuid NOT NULL,
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    "token"      bytea NOT NULL,
    "expires_at" timestamptz NOT NULL,
    "uses"       bigint NOT NULL DEFAULT 0,
    "group_invitation_tokens" uuid NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "group_invitation_tokens_groups_invitation_tokens" FOREIGN KEY ("group_invitation_tokens") REFERENCES "groups" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS "group_invitation_tokens_token_key" ON "group_invitation_tokens" ("token");

DROP TABLE IF EXISTS "role_permissions";
DROP TABLE IF EXISTS "user_roles";
DROP TABLE IF EXISTS "roles";
