ALTER TABLE users ADD COLUMN search_key VARCHAR(510) GENERATED ALWAYS AS (CONCAT(first_name, last_name, last_name, first_name)) VIRTUAL;
