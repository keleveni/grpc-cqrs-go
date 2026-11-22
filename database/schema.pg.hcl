schema "gcook" {}

table "users" {
  schema = schema.gcook
  column "id" {
    # IdP の sub をそのまま user id に使う(ULID 固定ではない不透明文字列)
    type = varchar(255)
  }
  column "name" {
    type = varchar(255)
  }
  column "email" {
    type = varchar(255)
  }
  column "created_at" {
    type    = timestamp
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamp
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  index "users_email_key" {
    columns = [column.email]
    unique  = true
  }
}

table "recipes" {
  schema = schema.gcook
  column "id" {
    type = uuid
  }
  column "user_id" {
    type = varchar(255)
  }
  column "title" {
    type = varchar(255)
  }
  column "description" {
    type = varchar(255)
  }
  column "visibility" {
    type = enum.visibility
  }
  column "created_at" {
    type    = timestamp
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamp
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "recipes_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = NO_ACTION
    on_update   = NO_ACTION
  }
}

enum "visibility" {
  schema = schema.gcook
  values = ["public", "private", "restricted"]
}

table "relation_tuples" {
  schema = schema.gcook
  column "id" {
    type = uuid
  }
  column "object_type" {
    type = varchar(255)
  }
  column "object_id" {
    type = varchar(255)
  }
  column "relation" {
    type = varchar(255)
  }
  column "user_id" {
    type = varchar(255)
  }
  column "created_at" {
    type    = timestamp
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  unique "unique" {
    columns     = [
      column.object_type, 
      column.object_id, 
      column.relation, 
      column.user_id,
    ]
  }
}

# 読み取りモデル（projection）: 誰がレシピを閲覧できるか。
# 書き込み側（公開範囲の変更・共有）が、公開範囲と tuple から計算した結果を書く。読み取り側はこれを引くだけ。
# public のレシピは user_id = '*'（全員）の 1 行で表す。
table "recipe_viewers" {
  schema = schema.gcook
  column "recipe_id" {
    type = uuid
  }
  column "user_id" {
    type = varchar(255)
  }
  primary_key {
    columns = [column.recipe_id, column.user_id]
  }
  foreign_key "recipe_viewers_recipe_id_fkey" {
    columns     = [column.recipe_id]
    ref_columns = [table.recipes.column.id]
    on_delete   = CASCADE
    on_update   = NO_ACTION
  }
  # 一覧（ユーザーが閲覧できるレシピ）用
  index "recipe_viewers_user_id_recipe_id_idx" {
    columns = [column.user_id, column.recipe_id]
  }
}
