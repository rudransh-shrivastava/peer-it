-- Schema for peer-it database

CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    size INTEGER NOT NULL,
    max_chunk_size INTEGER NOT NULL,
    total_chunks INTEGER NOT NULL,
    hash TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,
    chunk_size INTEGER NOT NULL,
    chunk_hash TEXT NOT NULL,
    is_available INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS peers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip_address TEXT NOT NULL,
    port TEXT NOT NULL,
    UNIQUE(ip_address, port)
);

CREATE TABLE IF NOT EXISTS swarms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    peer_id INTEGER NOT NULL REFERENCES peers(id) ON DELETE CASCADE,
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    UNIQUE(peer_id, file_id)
);

-- Indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_files_hash ON files(hash);
CREATE INDEX IF NOT EXISTS idx_chunks_file_id ON chunks(file_id);
CREATE INDEX IF NOT EXISTS idx_chunks_file_id_index ON chunks(file_id, chunk_index);
CREATE INDEX IF NOT EXISTS idx_peers_ip_port ON peers(ip_address, port);
CREATE INDEX IF NOT EXISTS idx_swarms_file_id ON swarms(file_id);
CREATE INDEX IF NOT EXISTS idx_swarms_peer_id ON swarms(peer_id);
