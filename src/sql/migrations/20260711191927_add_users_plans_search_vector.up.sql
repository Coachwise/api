SET search_path TO public;

-- Full-text search over users (username + name) and plans (name). 'simple'
-- config tokenizes en + fa; generated + GIN-indexed like exercises/tags.
ALTER TABLE users ADD COLUMN search_vector tsvector GENERATED ALWAYS AS (
  to_tsvector('simple',
    coalesce(username, '') || ' ' ||
    coalesce(first_name, '') || ' ' ||
    coalesce(last_name, ''))
) STORED;
CREATE INDEX idx_users_search_vector ON users USING gin (search_vector);

ALTER TABLE plans ADD COLUMN search_vector tsvector GENERATED ALWAYS AS (
  to_tsvector('simple', coalesce(name, ''))
) STORED;
CREATE INDEX idx_plans_search_vector ON plans USING gin (search_vector);
