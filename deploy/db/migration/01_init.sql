-- Init database for postgres, create user and database

-- Create database with id and name
CREATE DATABASE feeds WITH OWNER postgres;

GRANT ALL PRIVILEGES ON DATABASE feeds TO postgres;

-- Create table to docker database with id and name

CREATE TABLE IF NOT EXISTS sites (
    domain TEXT PRIMARY KEY,
    name TEXT,
    inserted_at TIMESTAMP DEFAULT current_timestamp,
    last_crawled_at TIMESTAMP,
    last_crawl_status TEXT
);

CREATE TABLE IF NOT EXISTS pages (
    url TEXT PRIMARY KEY,
    site_domain TEXT REFERENCES sites,
    path TEXT,
    parent_url TEXT,
    depth INTEGER,
    html TEXT,
    text TEXT,
    images TEXT[],
    status TEXT,
    meta JSONB,
    inserted_at TIMESTAMP
--     PRIMARY KEY (site_domain, path)
);

CREATE TABLE IF NOT EXISTS crawls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_domain TEXT REFERENCES sites,
    seed_url TEXT,
    status TEXT,
    started_at TIMESTAMP DEFAULT current_timestamp,
    ended_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS links (
    from_url TEXT,
    to_url TEXT,
    crawl_id UUID,
    PRIMARY KEY (from_url, to_url)
);

CREATE TABLE IF NOT EXISTS recipes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT UNIQUE,
    title TEXT,
    author TEXT,
    publication TEXT,
    blurb TEXT,
    ingredients JSONB,
    method JSONB,
    serving JSONB,
    cooking_time INTEGER,
    prep_time INTEGER,
    note TEXT,
);

CREATE INDEX idx_recipes_ingredients_gin_path
    ON recipes
    USING GIN (ingredients jsonb_path_ops);

CREATE TABLE IF NOT EXISTS ingredients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    canonical_name TEXT,
    display_name TEXT,
    display_name_plural TEXT,
    aliases TEXT[]
);