-- name: AddOutboxEvent :exec
INSERT INTO outbox (id, type, payload)
VALUES ($1, $2, $3);

-- name: FetchUnprocessedEvents :many
SELECT * FROM outbox
WHERE processed_at IS NULL
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkEventProcessed :exec
UPDATE outbox
SET processed_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: MarkEventsProcessedBatch :exec
UPDATE outbox
SET processed_at = CURRENT_TIMESTAMP
WHERE id = ANY($1::uuid[]);