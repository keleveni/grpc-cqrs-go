package cursor

import (
	"errors"
	"testing"
	"uuid"
)

type pageCursor struct {
	Before uuid.UUID `json:"before"`
}

func TestEncodeDecode(t *testing.T) {
	in := pageCursor{Before: uuid.NewV7()}

	token, err := Encode(in)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, err := Decode[pageCursor](token)
	if err != nil || got != in {
		t.Fatalf("round trip: got (%v, %v), want %v", got, err, in)
	}
}

func TestDecode_Invalid(t *testing.T) {
	for _, token := range []string{
		"!!!",                 // base64url でない
		"YWJj",                // base64url だが JSON でない（"abc"）
		"eyJiZWZvcmUiOiJ4In0", // JSON だが before が UUID でない（{"before":"x"}）
	} {
		if _, err := Decode[pageCursor](token); !errors.Is(err, ErrInvalid) {
			t.Errorf("%q: want ErrInvalid, got %v", token, err)
		}
	}
}
