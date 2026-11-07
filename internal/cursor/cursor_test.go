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

// 空文字は空文字のまま（「次のページなし」「最初のページ」をそのまま表せる）。
func TestEmpty(t *testing.T) {
	if got := Encode(""); got != "" {
		t.Fatalf("Encode(\"\") = %q, want empty", got)
	}
	if got, err := Decode(""); err != nil || got != "" {
		t.Fatalf("Decode(\"\") = (%q, %v), want empty", got, err)
	}
}

func TestDecode_Invalid(t *testing.T) {
	if _, err := Decode("!!!"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}
