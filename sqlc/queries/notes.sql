-- name: CreateNote :one
insert into notes (id, owner_id, title, content)
values ($1, $2, $3, $4)
returning id, owner_id, title, content, created_at, updated_at;

-- name: GetNoteByID :one
select id, owner_id, title, content, created_at, updated_at
from notes
where id = $1;

-- name: GetNotesByOwnerID :many
select id, owner_id, title, content, created_at, updated_at
from notes
where owner_id = $1
order by created_at desc;

-- name: UpdateNote :one
update notes
set title = $2, content = $3, updated_at = now()
where id = $1
returning id, owner_id, title, content, created_at, updated_at;


-- name: DeleteNote :exec
delete from notes
where id = $1;
