package entity

import (
	"errors"
)

var ErrAlreadyExists = errors.New("user already exits")

type User struct {
	id    string
	name  string
	email string
}

func (u *User) ID() string    { return u.id }
func (u *User) Name() string  { return u.name }
func (u *User) Email() string { return u.email }

// NewUser は業務フィールドを受け取ってユーザーを構築する。
// id は呼び出し側(app 層)が用意する。IdP の sub をそのまま user id として使う
// （ULID とは限らない不透明な文字列）。
func NewUser(id, name, email string) *User {
	return &User{
		id:    id,
		name:  name,
		email: email,
	}
}
