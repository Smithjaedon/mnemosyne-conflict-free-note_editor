-- name: CreateUser :one
insert into users (id, email, username, hashed_password)
values ($1, $2, $3, $4)
returning id, email, username, hashed_password;

-- name: GetUserByUsername :one
select id, email, username, hashed_password
from users
where username = $1;

-- name: GetUserByID :one
select id, email, username, hashed_password
from users
where id = $1;