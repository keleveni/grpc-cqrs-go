package query

// Storage は読み取り側の永続化。ユースケースごとのファイル（get_recipe.go 等）で定義した
// 小さなインタフェースをまとめる。メソッド・読み取りモデル・ユースケースは同じファイルに 1:1 で置く。
type Storage interface {
	getRecipeStorage
}
