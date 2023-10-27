CREATE USER api WITH ENCRYPTED PASSWORD 'qwerty';
CREATE DATABASE person_enricher;
GRANT ALL PRIVILEGES ON DATABASE person_enricher TO api;

\connect person_enricher

CREATE TABLE persons (
    id serial PRIMARY KEY,
    name text NOT NULL,
    surname text NOT NULL,
    patronymic text,
    age integer,
    gender text,
    nationality text
);

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO api;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO api;
