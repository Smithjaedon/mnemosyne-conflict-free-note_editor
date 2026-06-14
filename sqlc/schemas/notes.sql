create table notes (
  id text primary key,
  owner_id text not null references users(id) on delete cascade,
  title text not null,
  content text not null,
  content_version int not null default 1,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);