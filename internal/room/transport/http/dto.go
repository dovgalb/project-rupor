package httproom

import (
	"encoding/json"
	"io"
	"time"

	"github.com/dovgalb/project-rupor/internal/room/domain"
	"github.com/dovgalb/project-rupor/internal/room/usecase"
)

type createRoomRequest struct {
	Name string `json:"name"`
}

type roomResponse struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"ownerId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type roomWithRoleResponse struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"ownerId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	Role      string    `json:"role"`
}

type listRoomsResponse struct {
	Items []roomWithRoleResponse `json:"items"`
}

type memberResponse struct {
	UserID   string    `json:"userId"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joinedAt"`
}

type listMembersResponse struct {
	Items []memberResponse `json:"items"`
}

type inviteResponse struct {
	Code      string    `json:"code"`
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
}

func jsonDecode(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

func jsonEncode(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}

func roomToResponse(r *domain.Room) roomResponse {
	return roomResponse{
		ID:        r.ID().String(),
		OwnerID:   r.OwnerID().String(),
		Name:      r.Name().String(),
		CreatedAt: r.CreatedAt(),
	}
}

func roomWithRoleToResponse(rwr usecase.RoomWithRole) roomWithRoleResponse {
	return roomWithRoleResponse{
		ID:        rwr.Room.ID().String(),
		OwnerID:   rwr.Room.OwnerID().String(),
		Name:      rwr.Room.Name().String(),
		CreatedAt: rwr.Room.CreatedAt(),
		Role:      rwr.Role.String(),
	}
}

func membershipToResponse(m *domain.Membership) memberResponse {
	return memberResponse{
		UserID:   m.UserID().String(),
		Role:     m.Role().String(),
		JoinedAt: m.JoinedAt(),
	}
}

func inviteToResponse(i *domain.Invite) inviteResponse {
	return inviteResponse{
		Code:      i.Code().String(),
		CreatedBy: i.CreatedBy().String(),
		CreatedAt: i.CreatedAt(),
	}
}
