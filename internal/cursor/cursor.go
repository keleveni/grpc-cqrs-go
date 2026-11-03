// Package cursor は、ページングの起点などをクライアント向けの不透明なトークンに変換する。
//
// トークンは JSON を base64url にしただけで、暗号化も署名もしない。デコードすれば中身は読めるが、
// 「クライアントは中身を解釈・生成しない」という API の約束（AIP-158 の opaque）を守ってもらうためのもの。
// 改ざんされても困らない値（閲覧条件を毎回サーバーで判定している一覧の起点など）にだけ使う。
package cursor

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

// ErrInvalid はトークンが不正なとき（base64url / JSON として読めない）。
var ErrInvalid = errors.New("invalid cursor")

// Encode は v を不透明なトークンにする。
func Encode[T any](v T) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// Decode はトークンを T に戻す。読めないトークンは ErrInvalid を返す。
func Decode[T any](token string) (T, error) {
	var v T

	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return v, ErrInvalid
	}

	if err := json.Unmarshal(raw, &v); err != nil {
		return v, ErrInvalid
	}

	return v, nil
}
