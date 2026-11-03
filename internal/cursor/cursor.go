// Package cursor は、ページングの起点などの文字列をクライアント向けの不透明なトークンに変換する。
//
// トークンは base64url にしただけで、暗号化も署名もしない。デコードすれば中身は読めるが、
// 「クライアントは中身を解釈・生成しない」という API の約束（AIP-158 の opaque）を守ってもらうためのもの。
// 改ざんされても困らない値（閲覧条件を毎回サーバーで判定している一覧の起点など）にだけ使う。
package cursor

import (
	"encoding/base64"
	"errors"
)

// ErrInvalid はトークンが不正なとき（base64url として読めない）。
var ErrInvalid = errors.New("invalid cursor")

// Encode は s を不透明なトークンにする。
func Encode(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

// Decode はトークンを元の文字列に戻す。読めないトークンは ErrInvalid を返す。
func Decode(token string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", ErrInvalid
	}

	return string(raw), nil
}
