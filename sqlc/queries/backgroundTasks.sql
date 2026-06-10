-- name: CleanUpOldNotes :exec
delete from notes
where created_at < now() - interval '10 minutes'
and content = '';