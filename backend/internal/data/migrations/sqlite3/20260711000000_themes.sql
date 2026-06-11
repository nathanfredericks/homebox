-- +goose Up
create table if not exists themes
(
    id                uuid     not null
        primary key,
    created_at        datetime not null,
    updated_at        datetime not null,
    name              text     not null,
    colors            json     not null,
    radius            text default '0.5rem',
    font_sans         text default '',
    font_mono         text default '',
    branding          json     not null,
    nav_logo_path     text default '',
    sidebar_logo_path text default '',
    login_icon_path   text default ''
);
