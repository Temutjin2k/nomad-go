begin;

DROP TABLE IF EXISTS location_history;
DROP TABLE IF EXISTS driver_sessions;

DROP INDEX IF EXISTS idx_drivers_status;

DROP TABLE IF EXISTS drivers;

DROP TABLE IF EXISTS "driver_status";

commit;