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

-- name: DeleteRecipeViewers :exec
DELETE FROM recipe_viewers WHERE recipe_id = $1;

-- name: CreateRecipeViewers :exec
INSERT INTO recipe_viewers (recipe_id, user_id)
SELECT sqlc.arg('recipe_id')::uuid, unnest(sqlc.arg('user_ids')::text[]);

-- 読み取り側は、書き込み側が計算した recipe_viewers（projection）を引くだけで、閲覧のルールは持たない。
-- user_ids には、ユーザー本人と「全員」（'*'。public のレシピの閲覧者）を渡す。

-- name: GetViewableRecipe :one
-- 閲覧できるときだけレシピと作成者を返す（存在しない・閲覧できないときは行なし）。
-- recipes.user_id は NOT NULL かつ users への外部キー（ON DELETE NO ACTION）なので、
-- INNER JOIN でレシピの行が落ちることはない。
SELECT r.id, r.user_id, r.title, r.description, r.visibility, r.created_at, r.updated_at,
       u.name AS user_name, u.email AS user_email
FROM recipes r
JOIN users u ON r.user_id = u.id
WHERE r.id = sqlc.arg('id')
  AND EXISTS (
    SELECT 1 FROM recipe_viewers v
    WHERE v.recipe_id = r.id AND v.user_id = ANY(sqlc.arg('user_ids')::text[])
  );

-- name: ListViewableRecipes :many
-- 閲覧できるレシピを新しい順（id は UUIDv7 なので id の降順）に返す。
-- before より古いレシピだけを返す（最初のページは uuid の最大値を渡す）。
SELECT r.id, r.user_id, r.title, r.description, r.visibility, r.created_at, r.updated_at
FROM recipes r
WHERE r.id < sqlc.arg('before')
  AND EXISTS (
    SELECT 1 FROM recipe_viewers v
    WHERE v.recipe_id = r.id AND v.user_id = ANY(sqlc.arg('user_ids')::text[])
  )
ORDER BY r.id DESC
LIMIT sqlc.arg('page_limit');
