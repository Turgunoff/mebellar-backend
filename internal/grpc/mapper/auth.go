package mapper

import (
	"mebellar-backend/models"
	"mebellar-backend/pkg/pb"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// ToPBUser converts domain User to protobuf User.
func ToPBUser(u *models.User) *pb.User {
	if u == nil {
		return nil
	}
	return &pb.User{
		Id:          u.ID,
		FullName:    u.FullName,
		Phone:       u.Phone,
		Email:       u.Email,
		AvatarUrl:   u.AvatarURL,
		Role:        u.Role,
		OnesignalId: u.OneSignalID,
		HasPin:      u.HasPin,
		CreatedAt:   timestamppb.New(u.CreatedAt),
		UpdatedAt:   timestamppb.New(u.UpdatedAt),
	}
}
