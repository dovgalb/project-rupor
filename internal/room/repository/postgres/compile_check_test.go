package postgres_test

import (
	chuc "github.com/dovgalb/project-rupor/internal/channel/usecase"
	pg "github.com/dovgalb/project-rupor/internal/room/repository/postgres"
	roomuc "github.com/dovgalb/project-rupor/internal/room/usecase"
)

var (
	_ roomuc.RoomRepository       = (*pg.RoomRepository)(nil)
	_ roomuc.MembershipRepository = (*pg.MembershipRepository)(nil)
	_ roomuc.InviteRepository     = (*pg.InviteRepository)(nil)
	_ roomuc.InviteCodeGenerator  = (*pg.Base32CodeGen)(nil)
	_ chuc.MembershipQuery        = (*pg.MembershipQueryAdapter)(nil)
)
