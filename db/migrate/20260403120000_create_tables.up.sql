CREATE DATABASE IF NOT EXISTS sample;
USE sample;

CREATE TABLE IF NOT EXISTS `groups` (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name        VARCHAR(255)    NOT NULL,
  description TEXT            NOT NULL,
  deleted_at  DATETIME        NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE INDEX idx_groups_active ON `groups`(deleted_at, id);

CREATE TABLE IF NOT EXISTS users (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  first_name VARCHAR(255)    NOT NULL,
  last_name  VARCHAR(255)    NOT NULL,
  created_at DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME        NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE INDEX idx_users_active ON users(deleted_at, id);

CREATE TABLE IF NOT EXISTS group_members (
  id       BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  group_id BIGINT UNSIGNED NOT NULL,
  user_id  BIGINT UNSIGNED NOT NULL,
  CONSTRAINT fk_group_members_group_id  FOREIGN KEY (group_id) REFERENCES `groups`(id),
  CONSTRAINT fk_group_members_user_id   FOREIGN KEY (user_id)  REFERENCES users(id),
  CONSTRAINT uq_group_members_group_user UNIQUE (group_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- idx_group_members_group_id は uq_group_members_group_user の leftmost prefix と重複するため不要
CREATE INDEX idx_group_members_user_id ON group_members(user_id);
