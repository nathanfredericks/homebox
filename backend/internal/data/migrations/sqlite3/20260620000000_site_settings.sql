-- +goose Up
create table if not exists site_settings
(
    id         uuid     not null
        primary key,
    created_at datetime not null,
    updated_at datetime not null,
    key        text     not null,
    value      json     not null
);

create unique index if not exists site_settings_key_key
    on site_settings (key);
