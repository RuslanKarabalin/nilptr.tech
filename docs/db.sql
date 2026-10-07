create type nav_place as enum ('header', 'footer');
create type post_status as enum ('draft', 'unlisted', 'published');
create type comment_status as enum ('pending', 'approved', 'rejected');

create table users (
  id            uuid primary key,
  login         text not null unique,
  password_hash text not null
);

create table refresh_tokens (
  id            uuid primary key,
  user_id       uuid not null references users(id) on delete cascade,
  token_hash    text not null unique,
  family_id     uuid not null,
  user_agent    text not null,
  ip            inet not null,
  created_at    timestamptz not null default now(),
  last_used_at  timestamptz not null default now(),
  expires_at    timestamptz not null,
  revoked_at    timestamptz
);

create index refresh_tokens_family_id_idx on refresh_tokens (family_id);

create table posts (
  id            uuid primary key,
  slug          text not null unique,
  title         text not null,
  body          text not null,
  status        post_status not null default 'draft',
  created_at    timestamptz not null default now(),
  updated_at    timestamptz not null default now(),
  published_at  timestamptz
);

create table pages (
  id          uuid primary key,
  slug        text not null unique,
  title       text not null,
  body        text not null,
  updated_at  timestamptz not null default now()
);

create table nav_items (
  id        bigserial primary key,
  place     nav_place not null,
  label     text not null,
  url       text not null,
  position  int not null
);

create table files (
  id            uuid primary key,
  name          text not null,
  content_type  text not null,
  size          bigint not null,
  s3_key        text not null,
  created_at    timestamptz not null default now()
);

create table comments (
  id          bigserial primary key,
  post_id     bigint not null references posts(id) on delete cascade,
  author      text,
  body        text not null,
  status      comment_status not null default 'pending',
  ip_hash     text not null unique,
  created_at  timestamptz not null default now()
);

create table page_views (
  id          bigserial primary key,
  path        text not null,
  created_at  timestamptz not null default now()
);

create index page_views_created_at_idx on page_views (created_at);
