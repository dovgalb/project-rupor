// DTO и ViewModel для домена channels.
// Источник: docs/3_5_frontend/06-api-integration.md (REST: Channels).
// Без runtime-кода: только TS-типы.

export type ChannelId = string;
export type ChannelKind = "text" | "voice";

// === REST DTO (camelCase, из бэка) ===

export type ChannelDto = {
  id: string;
  roomId: string;
  name: string;
  kind: ChannelKind;
  createdAt: string;
};

export type CreateChannelRequest = {
  name: string;
  kind: ChannelKind;
};

export type ListChannelsResponse = { items: ChannelDto[] };

// === ViewModel = DTO (без переименований) ===

export type Channel = ChannelDto;
