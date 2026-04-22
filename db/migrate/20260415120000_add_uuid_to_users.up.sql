ALTER TABLE users
  ADD COLUMN uuid VARCHAR(36) NULL AFTER id;

-- NOTE: This UPDATE is intentionally included in the migration to backfill existing rows
-- before the NOT NULL constraint is applied. Separating this into a seed file would break
-- the migration sequence since golang-migrate runs all migrations before seed files.
UPDATE users SET uuid = UUID() WHERE uuid IS NULL;

ALTER TABLE users
  MODIFY COLUMN uuid VARCHAR(36) NOT NULL,
  ADD UNIQUE KEY uq_users_uuid (uuid);
