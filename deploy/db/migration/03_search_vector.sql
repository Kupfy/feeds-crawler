CREATE EXTENSION IF NOT EXISTS unaccent;

CREATE OR REPLACE FUNCTION immutable_unaccent(text)
RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT AS $$
    SELECT unaccent($1)
$$;

CREATE OR REPLACE FUNCTION normalise_text(input text)
RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT AS $$
    SELECT immutable_unaccent(
       regexp_replace(input, '[''`'']', '', 'g')  -- strip apostrophes/backticks
   )
$$;

CREATE OR REPLACE FUNCTION ingredients_to_text(ingredients jsonb)
RETURNS text
LANGUAGE sql IMMUTABLE AS $$
    SELECT string_agg(normalise_text(elem->>'name'), ' ')
    FROM jsonb_array_elements(ingredients) AS elem
$$;

ALTER TABLE recipes ADD COLUMN search_vector tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(author, '')), 'B') ||
    setweight(to_tsvector('english', coalesce(publication, '')), 'C') ||
    setweight(to_tsvector('english', coalesce(ingredients_to_text(ingredients), '')), 'B')
) STORED;

CREATE INDEX idx_recipes_search ON recipes USING GIN(search_vector);