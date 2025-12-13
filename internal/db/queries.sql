-- Queries for peer-it database

-- File queries
-- name: CreateFile :one
INSERT INTO files (name, size, max_chunk_size, total_chunks, hash, created_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetFileByHash :one
SELECT * FROM files WHERE hash = ? LIMIT 1;

-- name: GetFileByID :one
SELECT * FROM files WHERE id = ? LIMIT 1;

-- name: GetAllFiles :many
SELECT * FROM files;

-- name: GetFileNameByHash :one
SELECT name FROM files WHERE hash = ? LIMIT 1;

-- Chunk queries
-- name: CreateChunk :one
INSERT INTO chunks (file_id, chunk_index, chunk_size, chunk_hash, is_available)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetChunk :one
SELECT c.* FROM chunks c
JOIN files f ON c.file_id = f.id
WHERE f.hash = ? AND c.chunk_index = ?
LIMIT 1;

-- name: GetChunksByFileHash :many
SELECT c.* FROM chunks c
JOIN files f ON c.file_id = f.id
WHERE f.hash = ?
ORDER BY c.chunk_index;

-- name: GetChunksByFileID :many
SELECT * FROM chunks WHERE file_id = ? ORDER BY chunk_index;

-- name: MarkChunkAvailable :exec
UPDATE chunks SET is_available = 1
WHERE file_id = (SELECT id FROM files WHERE hash = ?)
AND chunk_index = ?;

-- Peer queries
-- name: CreatePeer :one
INSERT INTO peers (ip_address, port)
VALUES (?, ?)
RETURNING *;

-- name: GetPeerByIPPort :one
SELECT * FROM peers WHERE ip_address = ? AND port = ? LIMIT 1;

-- name: DeletePeerByIPPort :exec
DELETE FROM peers WHERE ip_address = ? AND port = ?;

-- name: DeleteAllPeers :exec
DELETE FROM peers;

-- name: GetPeersByFileHash :many
SELECT p.* FROM peers p
JOIN swarms s ON p.id = s.peer_id
JOIN files f ON s.file_id = f.id
WHERE f.hash = ?;

-- Swarm queries
-- name: CreateSwarm :one
INSERT INTO swarms (peer_id, file_id)
VALUES (?, ?)
RETURNING *;

-- name: AddPeerToSwarm :exec
INSERT INTO swarms (peer_id, file_id)
SELECT p.id, f.id
FROM peers p, files f
WHERE p.ip_address = ? AND p.port = ? AND f.hash = ?;

-- name: DeleteAllSwarms :exec
DELETE FROM swarms;
