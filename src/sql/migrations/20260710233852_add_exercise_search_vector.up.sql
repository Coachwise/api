SET search_path TO public;

-- Full-text search over exercises. 'simple' config (no stemming) tokenizes both
-- English and Persian reasonably; indexes name/description + their i18n values +
-- slug so search works whatever language the user typed.
ALTER TABLE exercises ADD COLUMN search_vector tsvector GENERATED ALWAYS AS (
  to_tsvector('simple',
    coalesce(name, '') || ' ' ||
    coalesce(description, '') || ' ' ||
    coalesce(name_i18n->>'en', '') || ' ' ||
    coalesce(name_i18n->>'fa', '') || ' ' ||
    coalesce(description_i18n->>'en', '') || ' ' ||
    coalesce(description_i18n->>'fa', '') || ' ' ||
    coalesce(replace(slug, '-', ' '), '')
  )
) STORED;

CREATE INDEX idx_exercises_search_vector ON exercises USING gin (search_vector);
