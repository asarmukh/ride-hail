begin;

-- User roles enumeration
create table roles("value" text not null primary key);
insert into "roles" ("value") values ('PASSENGER'), ('DRIVER'), ('ADMIN');

-- User status enumeration
create table user_status("value" text not null primary key);
insert into "user_status" ("value") values ('ACTIVE'), ('INACTIVE'), ('BANNED');

-- Main users table
create table users (
    id uuid primary key default gen_random_uuid(),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    email varchar(100) unique not null,
    role text references "roles"(value) not null,
    status text references "user_status"(value) not null default 'ACTIVE',
    password_hash text not null,
    attrs jsonb default '{}'::jsonb
);

commit;