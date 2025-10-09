begin;

-- Ride status enumeration
create table ride_status("value" text not null primary key);
insert into "ride_status" ("value") 
values ('REQUESTED'), ('MATCHED'), ('EN_ROUTE'), ('ARRIVED'),
       ('IN_PROGRESS'), ('COMPLETED'), ('CANCELLED');

-- Ride type enumeration
create table vehicle_type("value" text not null primary key);
insert into "vehicle_type" ("value") values ('ECONOMY'), ('PREMIUM'), ('XL');

-- Coordinates table
create table coordinates (
    id uuid primary key default gen_random_uuid(),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    entity_id uuid not null,
    entity_type varchar(20) not null check (entity_type in ('driver', 'passenger')),
    address text not null,
    latitude DOUBLE PRECISION not null check (latitude between -90 and 90),
    longitude DOUBLE PRECISION not null check (longitude between -180 and 180),    
    fare_amount decimal(10,2) check (fare_amount >= 0),
    distance_km decimal(8,2) check (distance_km >= 0),
    duration_minutes integer check (duration_minutes >= 0),
    is_current boolean default true
);

create index idx_coordinates_entity on coordinates(entity_id, entity_type);
create index idx_coordinates_current on coordinates(entity_id, entity_type) where is_current = true;

-- Main rides table
create table rides (
    id uuid primary key default gen_random_uuid(),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    ride_number varchar(50) unique not null,
    passenger_id uuid not null references users(id),
    driver_id uuid references users(id),
    vehicle_type text references "vehicle_type"(value),
    status text references "ride_status"(value),
    priority integer default 1 check (priority between 1 and 10),
    requested_at timestamptz default now(),
    matched_at timestamptz,
    arrived_at timestamptz,
    started_at timestamptz,
    completed_at timestamptz,
    cancelled_at timestamptz,
    cancellation_reason text,
    estimated_fare decimal(10,2),
    final_fare decimal(10,2),
    pickup_coordinate_id uuid references coordinates(id),
    destination_coordinate_id uuid references coordinates(id)
);

create index idx_rides_status on rides(status);

-- Event type enumeration
create table ride_event_type("value" text not null primary key);
insert into "ride_event_type" ("value")
values ('RIDE_REQUESTED'), ('DRIVER_MATCHED'), ('DRIVER_ARRIVED'),
       ('RIDE_STARTED'), ('RIDE_COMPLETED'), ('RIDE_CANCELLED'),
       ('STATUS_CHANGED'), ('LOCATION_UPDATED'), ('FARE_ADJUSTED');

-- Event sourcing table
create table ride_events (
    id uuid primary key default gen_random_uuid(),
    created_at timestamptz not null default now(),
    ride_id uuid references rides(id) not null,
    event_type text references "ride_event_type"(value),
    event_data jsonb not null
);

commit;
