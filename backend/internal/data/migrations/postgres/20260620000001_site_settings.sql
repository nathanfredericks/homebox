-- +goose Up
-- Create "site_settings" table
CREATE TABLE IF NOT EXISTS "site_settings" (
    "id" uuid NOT NULL,
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    "key" character varying NOT NULL,
    "value" jsonb NOT NULL,
    PRIMARY KEY ("id")
);
-- Create index "site_settings_key_key" to table: "site_settings"
CREATE UNIQUE INDEX IF NOT EXISTS "site_settings_key_key" ON "site_settings" ("key");
