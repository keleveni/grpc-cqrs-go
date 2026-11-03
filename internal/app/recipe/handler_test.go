package recipe

import (
	"errors"
	"testing"
	"uuid"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/cursor"
	pb "github.com/kodaikumatani/grpc-cqrs-go/pkg/pb/recipe/v1"
)

// proto に定義されたすべての値（UNSPECIFIED 以外）が往復変換できること。
// proto に値を足して変換を直し忘れるとここで落ちる。
func TestVisibilityRoundTrip(t *testing.T) {
	for n := range pb.Visibility_name {
		v := pb.Visibility(n)
		if v == pb.Visibility_VISIBILITY_UNSPECIFIED {
			continue
		}

		d, err := visibilityFromPb(v)
		if err != nil {
			t.Errorf("%v: want domain value, got %v", v, err)
			continue
		}
		if got := visibilityToPb(d); got != v {
			t.Errorf("%v: round trip got %v", v, got)
		}
	}
}

func TestVisibilityFromPb_Invalid(t *testing.T) {
	for _, v := range []pb.Visibility{pb.Visibility_VISIBILITY_UNSPECIFIED, pb.Visibility(99)} {
		if _, err := visibilityFromPb(v); !errors.Is(err, entity.ErrInvalidVisibility) {
			t.Errorf("%v: want ErrInvalidVisibility, got %v", v, err)
		}
	}
}

func TestVisibilityToPb_Unknown(t *testing.T) {
	if got := visibilityToPb(entity.Visibility{}); got != pb.Visibility_VISIBILITY_UNSPECIFIED {
		t.Errorf("want UNSPECIFIED, got %v", got)
	}
}

func TestPageToken(t *testing.T) {
	t.Run("次のページの起点は往復できる", func(t *testing.T) {
		next := uuid.NewV7()

		token, err := encodePageToken(next)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}

		got, err := decodePageToken(token)
		if err != nil || got != next {
			t.Fatalf("round trip: got (%v, %v), want %v", got, err, next)
		}
	})

	t.Run("次のページがなければ空のトークン", func(t *testing.T) {
		if got, err := encodePageToken(uuid.Nil()); err != nil || got != "" {
			t.Fatalf("want empty token, got (%q, %v)", got, err)
		}
	})

	t.Run("空のトークンは最初のページ", func(t *testing.T) {
		got, err := decodePageToken("")
		if err != nil || got != uuid.Max() {
			t.Fatalf("got (%v, %v), want uuid.Max()", got, err)
		}
	})

	t.Run("不正なトークンは cursor.ErrInvalid", func(t *testing.T) {
		if _, err := decodePageToken("!!!"); !errors.Is(err, cursor.ErrInvalid) {
			t.Fatalf("want cursor.ErrInvalid, got %v", err)
		}
	})
}
