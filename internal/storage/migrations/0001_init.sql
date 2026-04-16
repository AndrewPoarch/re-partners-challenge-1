CREATE TABLE IF NOT EXISTS pack_sizes (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    size  INTEGER NOT NULL UNIQUE CHECK (size > 0)
);

CREATE INDEX IF NOT EXISTS idx_pack_sizes_size ON pack_sizes(size);
