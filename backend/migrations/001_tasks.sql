CREATE TABLE IF NOT EXISTS tasks (
    id       SERIAL PRIMARY KEY,
    title    TEXT    NOT NULL,
    meta     TEXT    NOT NULL DEFAULT '',
    priority TEXT    NOT NULL DEFAULT 'medium'
             CHECK (priority IN ('high', 'medium', 'gentle')),
    done     BOOLEAN NOT NULL DEFAULT FALSE
);
