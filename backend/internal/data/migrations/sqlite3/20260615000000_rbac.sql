-- +goose Up
-- +goose no transaction
-- RBAC overhaul: collections belong to the site, not to users. Access is
-- granted through roles (shown as "Groups" in the UI) that carry a granular
-- per-collection permission matrix. Group membership (user_groups) and the
-- invitation system are removed. Existing group owners and legacy superusers
-- are backfilled into the seeded Super Admin role; all other users start with
-- no roles and must be granted access by an admin.
PRAGMA foreign_keys=OFF;

-- 1. Roles ("Groups" in the UI).
create table if not exists roles
(
    id             uuid     not null
        primary key,
    created_at     datetime not null,
    updated_at     datetime not null,
    name           text     not null,
    description    text,
    is_super_admin boolean  not null default false
);

create unique index if not exists roles_name_key
    on roles (name);

-- 2. Permission matrix rows. collection_id NULL = all collections (or the
-- site scope for site-level sections).
create table if not exists role_permissions
(
    id            uuid     not null
        primary key,
    created_at    datetime not null,
    updated_at    datetime not null,
    section       text     not null,
    can_view      boolean  not null default false,
    can_create    boolean  not null default false,
    can_edit      boolean  not null default false,
    can_delete    boolean  not null default false,
    collection_id uuid
        constraint role_permissions_groups_role_permissions
            references groups
            on delete cascade,
    role_id       uuid     not null
        constraint role_permissions_roles_permissions
            references roles
            on delete cascade
);

create unique index if not exists rolepermission_role_id_section_collection_id
    on role_permissions (role_id, section, collection_id);

-- 3. User <-> Role assignment.
create table if not exists user_roles
(
    user_id uuid not null
        constraint user_roles_user_id
            references users
            on delete cascade,
    role_id uuid not null
        constraint user_roles_role_id
            references roles
            on delete cascade,
    primary key (user_id, role_id)
);

-- 4. Seed the Super Admin role (fixed UUID; evaluation short-circuits on
-- is_super_admin so it needs no permission rows).
INSERT INTO roles (id, created_at, updated_at, name, description, is_super_admin)
SELECT '00000000-0000-0000-0000-000000000001', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
       'Super Admin', 'Full access to everything. This group cannot be edited or deleted.', true
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE is_super_admin = true);

-- 5. Backfill: group owners and legacy superusers become Super Admins.
INSERT INTO user_roles (user_id, role_id)
SELECT DISTINCT u.id, '00000000-0000-0000-0000-000000000001'
FROM users u
WHERE u.is_superuser = true
   OR u.superuser = true
   OR EXISTS (SELECT 1 FROM user_groups ug WHERE ug.user_id = u.id AND ug.role = 'owner');

-- 6. Drop membership + invitations.
DROP TABLE IF EXISTS user_groups;
DROP TABLE IF EXISTS group_invitation_tokens;

-- 7. Drop users.is_superuser / users.superuser via table rebuild (SQLite-safe).
CREATE TABLE users_new (
    id UUID NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    password TEXT,
    activated_on DATETIME,
    oidc_issuer TEXT,
    oidc_subject TEXT,
    default_group_id UUID,
    settings JSON DEFAULT '{}',
    PRIMARY KEY (id),
    CONSTRAINT users_groups_users_default FOREIGN KEY (default_group_id) REFERENCES groups(id) ON DELETE SET NULL
);

INSERT INTO users_new (
    id, created_at, updated_at, name, email, password,
    activated_on, oidc_issuer, oidc_subject, default_group_id, settings
)
SELECT
    id, created_at, updated_at, name, email, password,
    activated_on, oidc_issuer, oidc_subject, default_group_id, settings
FROM users;

DROP INDEX IF EXISTS users_email_key;
DROP INDEX IF EXISTS users_oidc_issuer_subject_key;

DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

CREATE UNIQUE INDEX IF NOT EXISTS users_email_key ON users(email);
CREATE UNIQUE INDEX IF NOT EXISTS users_oidc_issuer_subject_key ON users(oidc_issuer, oidc_subject);

PRAGMA foreign_keys=ON;

-- +goose Down
-- +goose no transaction
-- Best effort: membership data cannot be fully reconstructed. Users are
-- re-attached to their default group; Super Admins become its owner.
PRAGMA foreign_keys=OFF;

CREATE TABLE users_new (
    id UUID NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    password TEXT,
    is_superuser BOOLEAN NOT NULL DEFAULT false,
    superuser BOOLEAN NOT NULL DEFAULT false,
    activated_on DATETIME,
    oidc_issuer TEXT,
    oidc_subject TEXT,
    default_group_id UUID,
    settings JSON DEFAULT '{}',
    PRIMARY KEY (id),
    CONSTRAINT users_groups_users_default FOREIGN KEY (default_group_id) REFERENCES groups(id) ON DELETE SET NULL
);

INSERT INTO users_new (
    id, created_at, updated_at, name, email, password, is_superuser, superuser,
    activated_on, oidc_issuer, oidc_subject, default_group_id, settings
)
SELECT
    id, created_at, updated_at, name, email, password, false, false,
    activated_on, oidc_issuer, oidc_subject, default_group_id, settings
FROM users;

DROP INDEX IF EXISTS users_email_key;
DROP INDEX IF EXISTS users_oidc_issuer_subject_key;

DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

CREATE UNIQUE INDEX IF NOT EXISTS users_email_key ON users(email);
CREATE UNIQUE INDEX IF NOT EXISTS users_oidc_issuer_subject_key ON users(oidc_issuer, oidc_subject);

create table if not exists user_groups
(
    user_id  uuid not null
        constraint user_groups_user_id
            references users
            on delete cascade,
    group_id uuid not null
        constraint user_groups_group_id
            references groups
            on delete cascade,
    role     text not null default 'user',
    primary key (user_id, group_id)
);

INSERT INTO user_groups (user_id, group_id, role)
SELECT u.id,
       u.default_group_id,
       CASE
           WHEN EXISTS (SELECT 1
                        FROM user_roles ur
                                 JOIN roles r ON r.id = ur.role_id
                        WHERE ur.user_id = u.id
                          AND r.is_super_admin = true) THEN 'owner'
           ELSE 'user'
       END
FROM users u
WHERE u.default_group_id IS NOT NULL;

create table if not exists group_invitation_tokens
(
    id         uuid     not null
        primary key,
    created_at datetime not null,
    updated_at datetime not null,
    token      blob     not null,
    expires_at datetime not null,
    uses       integer  not null default 0,
    group_invitation_tokens uuid
        constraint group_invitation_tokens_groups_invitation_tokens
            references groups
            on delete cascade
);

create unique index if not exists group_invitation_tokens_token_key
    on group_invitation_tokens (token);

DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS roles;

PRAGMA foreign_keys=ON;
