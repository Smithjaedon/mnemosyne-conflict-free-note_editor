-- +goose Up
alter table notes
add column content_version int not null default 1;

-- +goose Down
alter table notes
drop column content_version;
