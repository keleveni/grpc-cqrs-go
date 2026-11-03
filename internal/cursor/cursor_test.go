package cursor

import (
	"errors"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	for _, in := range []string{"019a3b2c-0000-7000-8000-000000000000", "", "日本語 / +="} {
		got, err := Decode(Encode(in))
		if err != nil || got != in {
			t.Errorf("round trip %q: got (%q, %v)", in, got, err)
		}
	}
}

func TestDecode_Invalid(t *testing.T) {
	if _, err := Decode("!!!"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}
