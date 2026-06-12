-- name: GetSharedNotesByUserID :many
select n.id, n.title, n.content, n.created_at, n.updated_at, sn.permissions
from notes n
join share_notes sn on n.id = sn.note_id
where sn.user_id = $1;

-- name: GetSharedNoteByID :one
select n.id, n.title, n.content, n.created_at, n.updated_at, sn.permissions
from notes n
join share_notes sn on n.id = sn.note_id
where sn.user_id = $1 and n.id = $2;

-- name: GetNoteUsers :many
select u.id, u.email, u.username, sn.permissions
from users u
join share_notes sn on u.id = sn.user_id
where sn.note_id = $1;

-- name: AddNoteUser :one
insert into share_notes (user_id, note_id, permissions)
values ($1, $2, $3)
returning user_id, note_id, permissions;

-- name: RemoveNoteUser :exec
delete from share_notes where user_id = $1 and note_id = $2;

-- name: UpdateNoteUserPermissions :exec
update share_notes set permissions = $3 where user_id = $1 and note_id = $2;


-- name: GetSharedNoteByOwnerID :many
select sn.note_id
from share_notes sn
join notes n on n.id = sn.note_id
where n.owner_id = $1;