create table users (
  id text primary key,
  email text not null,
  username text not null unique,
  hashed_password text not null
);