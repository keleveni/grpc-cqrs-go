-- Create "recipe_viewers" table
CREATE TABLE "recipe_viewers" (
  "recipe_id" uuid NOT NULL,
  "user_id" character varying(255) NOT NULL,
  PRIMARY KEY ("recipe_id", "user_id"),
  CONSTRAINT "recipe_viewers_recipe_id_fkey" FOREIGN KEY ("recipe_id") REFERENCES "recipes" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "recipe_viewers_user_id_recipe_id_idx" to table: "recipe_viewers"
CREATE INDEX "recipe_viewers_user_id_recipe_id_idx" ON "recipe_viewers" ("user_id", "recipe_id");
-- 既存のレシピの閲覧者を埋める（一度きりのデータ移行。以降はアプリが書き込み時に更新する）
-- public は全員 / private は owner / restricted は viewer・editor・owner
INSERT INTO "recipe_viewers" ("recipe_id", "user_id")
SELECT r.id, '*' FROM "recipes" r WHERE r.visibility = 'public'
UNION
SELECT r.id, t.user_id
FROM "recipes" r
JOIN "relation_tuples" t ON t.object_type = 'recipe' AND t.object_id = r.id::text
WHERE (r.visibility = 'private' AND t.relation = 'owner')
   OR (r.visibility = 'restricted' AND t.relation IN ('viewer', 'editor', 'owner'));
