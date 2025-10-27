begin;

-- Driver status enumeration
create table "driver_status"("value" text not null primary key);
insert into "driver_status" ("value") values ('OFFLINE'), ('AVAILABLE'), ('BUSY'), ('EN_ROUTE');

-- Main drivers table
create table drivers (
    id uuid primary key references users(id),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    license_number varchar(50) unique not null,
    vehicle_type text references "vehicle_type"(value),
    vehicle_attrs jsonb,
    rating decimal(3,2) default 5.0 check (rating between 1.0 and 5.0),
    total_rides integer default 0 check (total_rides >= 0),
    total_earnings decimal(10,2) default 0 check (total_earnings >= 0),
    status text references "driver_status"(value),
    is_verified boolean default false
);

create index idx_drivers_status on drivers(status);

-- Driver sessions
create table driver_sessions (
    id uuid primary key default gen_random_uuid(),
    driver_id uuid references drivers(id) not null,
    started_at timestamptz not null default now(),
    ended_at timestamptz,
    total_rides integer default 0,
    total_earnings decimal(10,2) default 0
);

-- Location history
create table location_history (
    id uuid primary key default gen_random_uuid(),
    coordinate_id uuid references coordinates(id),
    driver_id uuid references drivers(id),
    latitude DOUBLE PRECISION not null check (latitude between -90 and 90),
    longitude DOUBLE PRECISION not null check (longitude between -180 and 180),
    accuracy_meters decimal(6,2),
    speed_kmh decimal(5,2),
    heading_degrees decimal(5,2) check (heading_degrees between 0 and 360),
    recorded_at timestamptz not null default now(),
    ride_id uuid references rides(id)
);

INSERT INTO users (id, email, role, status, password_hash)
VALUES (
    '550e8400-e29b-41d4-a716-446655440001',
    'passenger@example.com',
    'DRIVER',
    'ACTIVE',
    'test_hash_123'
);

commit;