ALTER TABLE `groups`
  ADD COLUMN updated_by BIGINT UNSIGNED NOT NULL,
  ADD CONSTRAINT fk_groups_updated_by FOREIGN KEY (updated_by) REFERENCES users(id);
