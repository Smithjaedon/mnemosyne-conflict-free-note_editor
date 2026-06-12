-- +goose Up
insert into share_notes (user_id, note_id, permissions)
select owner_id, id, 'owner' from notes
on conflict do nothing;

-- +goose Down
delete from share_notes where permissions = 'owner';
