package recipe

import (
	"errors"
	"testing"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
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

		d, err := entity.NewVisibility(visibilityFromPb(v))
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
		if _, err := entity.NewVisibility(visibilityFromPb(v)); !errors.Is(err, entity.ErrInvalidVisibility) {
			t.Errorf("%v: want ErrInvalidVisibility, got %v", v, err)
		}
	}
}

func TestVisibilityToPb_Unknown(t *testing.T) {
	if got := visibilityToPb(entity.Visibility{}); got != pb.Visibility_VISIBILITY_UNSPECIFIED {
		t.Errorf("want UNSPECIFIED, got %v", got)
	}
}
