begin;
-- Create a test passenger
INSERT INTO users (id, email, role, status, password_hash)
VALUES (
    '550e8400-e29b-41d4-a716-446655440001',
    'passenger@example.com',
    'PASSENGER',
    'ACTIVE',
    'test_hash_123'
);

commit;