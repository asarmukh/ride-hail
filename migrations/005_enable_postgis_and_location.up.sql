begin;

-- Make sure PostGIS is available
CREATE EXTENSION IF NOT EXISTS postgis;

-- Add geography (location) column to location_history
ALTER TABLE location_history ADD COLUMN location GEOGRAPHY(Point, 4326);

-- Backfill existing rows with Point geometry
UPDATE location_history
SET location = ST_SetSRID(ST_MakePoint(longitude::double precision, latitude::double precision), 4326)
WHERE location IS NULL;

-- Add spatial index
CREATE INDEX idx_location_history_location ON location_history USING GIST (location);

commit;