-- +goose Up
-- Create "themes" table
CREATE TABLE IF NOT EXISTS "themes" (
    "id" uuid NOT NULL,
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    "name" character varying NOT NULL,
    "colors" jsonb NOT NULL,
    "radius" character varying DEFAULT '0.5rem',
    "font_sans" character varying DEFAULT '',
    "font_mono" character varying DEFAULT '',
    "branding" jsonb NOT NULL,
    "nav_logo_path" character varying DEFAULT '',
    "sidebar_logo_path" character varying DEFAULT '',
    "login_icon_path" character varying DEFAULT '',
    PRIMARY KEY ("id")
);
