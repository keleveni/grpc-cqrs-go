-- 公開範囲 restricted と relation viewer をなくす（公開範囲は public / private、relation は owner / editor）。
-- 既存の restricted は private にする（見られる人が増えない側に倒す）
UPDATE "recipes" SET "visibility" = 'private' WHERE "visibility" = 'restricted';
-- viewer の共有は閲覧にも編集にも効かなくなるので消す
DELETE FROM "relation_tuples" WHERE "object_type" = 'recipe' AND "relation" = 'viewer';
-- private のレシピの閲覧者を owner だけにする（restricted から移したレシピの共有相手を外す）
DELETE FROM "recipe_viewers" v
USING "recipes" r
WHERE v.recipe_id = r.id
  AND r.visibility = 'private'
  AND NOT EXISTS (
    SELECT 1 FROM "relation_tuples" t
    WHERE t.object_type = 'recipe' AND t.object_id = r.id::text
      AND t.relation = 'owner' AND t.user_id = v.user_id
  );
-- enum から restricted を外す（Postgres は enum の値を削除できないので型を作り直す）
ALTER TYPE "visibility" RENAME TO "visibility_old";
CREATE TYPE "visibility" AS ENUM ('public', 'private');
ALTER TABLE "recipes" ALTER COLUMN "visibility" TYPE "visibility" USING "visibility"::text::"visibility";
DROP TYPE "visibility_old";
