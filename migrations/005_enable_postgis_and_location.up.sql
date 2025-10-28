begin;

-- Make sure PostGIS is available
CREATE EXTENSION IF NOT EXISTS postgis;

-- Add geography (location) column to coordinates
ALTER TABLE coordinates ADD COLUMN location GEOGRAPHY(Point, 4326);

ALTER TABLE location_history ADD COLUMN location GEOGRAPHY(Point, 4326);

-- Backfill existing rows with Point geometry
UPDATE coordinates
SET location = ST_SetSRID(ST_MakePoint(longitude::double precision, latitude::double precision), 4326)
WHERE location IS NULL;


UPDATE location_history
SET location = ST_SetSRID(ST_MakePoint(longitude::double precision, latitude::double precision), 4326)
WHERE location IS NULL;

-- Add spatial index
CREATE INDEX idx_coordinates_location ON coordinates USING GIST (location);

commit;