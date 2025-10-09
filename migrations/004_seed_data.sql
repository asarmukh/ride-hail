-- Seed lookup tables
INSERT INTO roles (value) VALUES ('PASSENGER'), ('DRIVER'), ('ADMIN');

INSERT INTO user_status (value)
VALUES ('ACTIVE'), ('INACTIVE'), ('BANNED');

INSERT INTO ride_status (value)
VALUES ('REQUESTED'), ('MATCHED'), ('EN_ROUTE'), ('ARRIVED'), ('IN_PROGRESS'), ('COMPLETED'), ('CANCELLED');

INSERT INTO vehicle_type (value)
VALUES ('ECONOMY'), ('PREMIUM'), ('XL');

INSERT INTO ride_event_type (value)
VALUES ('RIDE_REQUESTED'), ('DRIVER_MATCHED'), ('RIDE_STARTED'), ('RIDE_COMPLETED'), ('RIDE_CANCELLED');

-- Create a test passenger
INSERT INTO users (id, email, role, status, password_hash)
VALUES (
    '550e8400-e29b-41d4-a716-446655440001',
    'passenger@example.com',
    'PASSENGER',
    'ACTIVE',
    'test_hash_123'
);
