create table share_notes (
  user_id text not null references users(id) on delete cascade,
  note_id text not null references notes(id) on delete cascade,
  primary key (user_id, note_id),
  permissions text not null
);