# Manual QA: 2_1 rooms and channels

HTTP-запросы для ручного тестирования фичи PR-2 (комнаты + каналы):
- 7 эндпоинтов room (CreateRoom, GetRoom, ListUserRooms, DeleteRoom, ListMembers, RegenerateInvite, JoinByCode).
- 3 эндпоинта channel (CreateChannel, ListChannels, DeleteChannel).

Формат `.http` — совместим с VS Code REST Client и JetBrains HTTP Client.

## Подготовка стенда

```bash
make dc-up && make migrate-up && make run
```

В `.env` (или ENV) должны быть выставлены:
- `JWT_SECRET=...`
- `DATABASE_URL=...`

## Файлы

- `00_flow.http` — end-to-end сценарий: 2 пользователя, register → login → create room → invite → join → channel create/list/delete → delete room → CASCADE-проверка.
- `01_create_room.http` — POST /rooms: happy + 401 + 400 (ROOM-001) + 400 (ROOM-009).
- `02_get_list_rooms.http` — GET /rooms и GET /rooms/{id}: 200, 403, 404.
- `03_invite.http` — POST /rooms/{id}/invite: happy, regenerate (старый код отзывается), 403 ROOM-004.
- `04_join.http` — POST /rooms/join/{code}: happy, 409 ROOM-006, 400 ROOM-008, 404 ROOM-007.
- `05_members.http` — GET /rooms/{id}/members: 200, 403 ROOM-003.
- `06_channels.http` — text/voice create, 403 CHANNEL-007, 400 CHANNEL-002, 409 CHANNEL-004, list, delete.
- `07_delete_room.http` — DELETE /rooms/{id}: 204, 404, 403 ROOM-005.
- `99_smoke.http` — GET /api/v1/health.

## Переменные

В каждом файле наверху объявлены `@host` и переменные, которые нужно подставить руками
(например `@access_token_a`, `@room_id`, `@invite_code`). Получите их из шагов
`00_flow.http` либо отдельных сценариев `01..07`.

## Порядок прогона

1. `99_smoke.http` — убедиться, что сервер поднят.
2. `00_flow.http` — основной поток. После него у вас будут токены user A/B и room/invite/channel.
3. Дополнительные сценарии `01..07` — каждый файл проверяет один эндпоинт по всем веткам.
