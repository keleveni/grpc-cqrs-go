package recipe

import (
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
	pb "github.com/kodaikumatani/grpc-cqrs-go/pkg/pb/recipe/v1"
)

// visibilityFromPb は pb の Visibility をドメインの Visibility に変換する。
// UNSPECIFIED や未定義の値は ErrInvalidVisibility を返す。
func visibilityFromPb(v pb.Visibility) (entity.Visibility, error) {
	switch v {
	case pb.Visibility_VISIBILITY_PUBLIC:
		return entity.VisibilityPublic, nil
	case pb.Visibility_VISIBILITY_PRIVATE:
		return entity.VisibilityPrivate, nil
	case pb.Visibility_VISIBILITY_RESTRICTED:
		return entity.VisibilityRestricted, nil
	default:
		return entity.Visibility{}, entity.ErrInvalidVisibility
	}
}

// visibilityToPb はドメインの Visibility を pb の Visibility に変換する。
// ドメインの値は閉じた集合なので失敗しない（未知の値は UNSPECIFIED）。
func visibilityToPb(v entity.Visibility) pb.Visibility {
	switch v {
	case entity.VisibilityPublic:
		return pb.Visibility_VISIBILITY_PUBLIC
	case entity.VisibilityPrivate:
		return pb.Visibility_VISIBILITY_PRIVATE
	case entity.VisibilityRestricted:
		return pb.Visibility_VISIBILITY_RESTRICTED
	default:
		return pb.Visibility_VISIBILITY_UNSPECIFIED
	}
}
