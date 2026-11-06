-- name: CreateTuple :exec
INSERT INTO relation_tuples (id, object_type, object_id, relation, user_id)
VALUES ($1, $2, $3, $4, $5);

-- name: DeleteTuple :exec 
DELETE FROM relation_tuples WHERE id = $1;

-- name: ListRelations :many
SELECT id, object_type, object_id, relation, user_id, created_at
FROM relation_tuples
WHERE object_type = $1 AND object_id = $2 AND user_id = $3;

-- name: ListObjectIDsByRelations :many
-- user_id が relations のいずれかを持つ object_type の object_id を返す（一覧の絞り込み用）。
SELECT DISTINCT object_id
FROM relation_tuples
WHERE object_type = sqlc.arg('object_type')
  AND user_id = sqlc.arg('user_id')
  AND relation = ANY(sqlc.arg('relations')::text[]);
