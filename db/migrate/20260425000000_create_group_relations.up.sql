CREATE TABLE IF NOT EXISTS group_relations (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  parent_group_id BIGINT UNSIGNED NOT NULL,
  child_group_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_parent_child (parent_group_id, child_group_id),
  KEY idx_parent (parent_group_id),
  KEY idx_child (child_group_id),
  CONSTRAINT fk_parent_group FOREIGN KEY (parent_group_id) REFERENCES `groups`(id) ON DELETE CASCADE,
  CONSTRAINT fk_child_group FOREIGN KEY (child_group_id) REFERENCES `groups`(id) ON DELETE CASCADE
);
