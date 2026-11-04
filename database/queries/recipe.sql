-- name: CreateRecipe :exec
INSERT INTO recipes (id, user_id, title, description, visibility)
VALUES ($1, $2, $3, $4, $5);

-- name: GetRecipe :one
SELECT id, user_id, title, description, visibility
FROM recipes
WHERE id = $1;

-- name: UpdateRecipe :exec
UPDATE recipes
SET title = $2, description = $3, visibility = $4, updated_at = now()
WHERE id = $1;

-- 読み取り側の閲覧条件は、この 2 つのクエリ（GetViewableRecipe / ListViewableRecipes）だけが持つ。
-- 2 つの WHERE の閲覧条件は同じにする（変えるときは両方直す）:
--   public は誰でも / private は manage 権限 / restricted は view 権限
-- どの relation が manage / view 権限を与えるかは authz が決めるので、引数で受け取る。

-- name: GetViewableRecipe :one
-- user_id が閲覧できるときだけレシピと作成者を返す（存在しない・閲覧できないときは行なし）。
-- recipes.user_id は NOT NULL かつ users への外部キー（ON DELETE NO ACTION）なので、
-- INNER JOIN でレシピの行が落ちることはない。
SELECT r.id, r.user_id, r.title, r.description, r.visibility, r.created_at, r.updated_at,
       u.name AS user_name, u.email AS user_email
FROM recipes r
JOIN users u ON r.user_id = u.id
WHERE r.id = sqlc.arg('id')
  AND (
    r.visibility = 'public'
    OR EXISTS (
      SELECT 1
      FROM relation_tuples t
      WHERE t.object_type = 'recipe'
        AND t.object_id = r.id::text
        AND t.user_id = sqlc.arg('user_id')
        AND (
          (r.visibility = 'private' AND t.relation = ANY(sqlc.arg('manage_relations')::text[]))
          OR (r.visibility = 'restricted' AND t.relation = ANY(sqlc.arg('view_relations')::text[]))
        )
    )
  );

-- name: ListViewableRecipes :many
-- user_id が閲覧できるレシピを新しい順（id は UUIDv7 なので id の降順）に返す。
-- 閲覧条件は GetViewableRecipe と同じ。before より古いレシピだけを返す（最初のページは uuid の最大値を渡す）。
SELECT r.id, r.user_id, r.title, r.description, r.visibility, r.created_at, r.updated_at
FROM recipes r
WHERE r.id < sqlc.arg('before')
  AND (
    r.visibility = 'public'
    OR EXISTS (
      SELECT 1
      FROM relation_tuples t
      WHERE t.object_type = 'recipe'
        AND t.object_id = r.id::text
        AND t.user_id = sqlc.arg('user_id')
        AND (
          (r.visibility = 'private' AND t.relation = ANY(sqlc.arg('manage_relations')::text[]))
          OR (r.visibility = 'restricted' AND t.relation = ANY(sqlc.arg('view_relations')::text[]))
        )
    )
  )
ORDER BY r.id DESC
LIMIT sqlc.arg('page_limit');
