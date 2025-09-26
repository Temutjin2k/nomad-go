begin;
-- Event sourcing table for complete ride audit trail
drop table if exists ride_events;

-- Event type enumeration for audit trail
drop table if exists "ride_event_type";

-- Drop index for status queries
drop index if exists idx_rides_status;

-- Main rides table
drop table if exists rides;

-- Drop indexes for coordinates table
drop index if exists idx_coordinates_current;
drop index if exists idx_coordinates_entity;

-- Coordinates table for real-time location tracking
drop table if exists coordinates;

-- Ride type enumeration
drop table if exists "vehicle_type";

-- Ride status enumeration
drop table if exists "ride_status";

-- Main users table
drop table if exists users;

-- User status enumeration
drop table if exists "user_status";

-- User roles enumeration
drop table if exists "roles";

commit;