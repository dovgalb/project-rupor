# Manual QA: WebSocket-чат через browser console

**Prerequisite:** сервер запущен (`make run`), есть валидный access token и channel_id из 00_flow.http.

## Сценарий 1: connect + subscribe + message.send (happy path)

1. Открой Chrome DevTools → Console.
2. Выполни:

```js
const token = "<вставь access_token_a>";
const channelId = "<вставь channel_id>";
const ws = new WebSocket(`ws://localhost:8080/api/v1/ws?token=${token}`);

ws.onopen = () => console.log("WS open");
ws.onmessage = (e) => console.log("WS frame:", JSON.parse(e.data));
ws.onclose = (e) => console.log("WS close", e.code, e.reason);
ws.onerror = (e) => console.log("WS error", e);
```

3. После `WS open`, подписываемся на канал:

```js
ws.send(JSON.stringify({ type: "subscribe", channel_id: channelId }));
// Ожидаем фрейм: {type:"subscribed", data:{channel_id:"..."}}
```

4. Отправляем сообщение:

```js
ws.send(JSON.stringify({ type: "message.send", channel_id: channelId, text: "Привет!" }));
// Ожидаем 2 фрейма (порядок может варьироваться):
//   {type:"message.sent", data:{id, channel_id, created_at}} — ack отправителю
//   {type:"message.new",  data:{id, channel_id, author_id, text, created_at}} — broadcast
```

5. Проверь, что REST-история отдаёт это сообщение:

```bash
curl -s "http://localhost:8080/api/v1/channels/$channelId/messages" \
  -H "Authorization: Bearer $token" | jq .
```

## Сценарий 2: второй клиент видит сообщения первого

1. В другой вкладке DevTools открой соединение с токеном Bob'а на тот же канал.
2. Подпишись `subscribe` на канал.
3. В первой вкладке отправь `message.send`.
4. Во второй вкладке должен прийти фрейм `message.new` с тем же текстом.

## Сценарий 3: error CHAT-004 (не член)

```js
const wsX = new WebSocket(`ws://localhost:8080/api/v1/ws?token=<token-не-члена>`);
wsX.onmessage = (e) => console.log(JSON.parse(e.data));
wsX.onopen = () => wsX.send(JSON.stringify({type:"subscribe", channel_id:"<channel-where-not-member>"}));
// Ожидаем: {type:"error", data:{code:"CHAT-004", message:"access denied: not a room member"}}
```

## Сценарий 4: error CHAT-003 (voice channel)

1. Создай voice-канал через REST (`kind: "voice"`).
2. Подпишись и попробуй `message.send` в этот канал.
3. Ожидаем: `{type:"error", data:{code:"CHAT-003", message:"channel is not text"}}`

## Сценарий 5: 401 до апгрейда

1. Подключись без токена:

```js
const wsNoTok = new WebSocket("ws://localhost:8080/api/v1/ws");
wsNoTok.onerror = () => console.log("expected 401");
```

Browser не даст обычно увидеть HTTP-401 явно (WebSocket-handshake падает). Через curl:

```bash
curl -i "http://localhost:8080/api/v1/ws"
# HTTP/1.1 401 Unauthorized
# {"error":{"code":"AUTH-010","message":"access token invalid"}}
```

## Сценарий 6: server shutdown closes with 1001

1. Подключись через WS.
2. `Ctrl+C` на сервере.
3. Клиентский `onclose` должен сработать с `e.code = 1001 (Going Away)`.

## Сценарий 7: token не утекает в access-лог

1. Подключись через WS с любым токеном.
2. Глянь stdout сервера — путь должен быть `/api/v1/ws?token=REDACTED`, а не реальный токен.

## Post-test чек-лист
- [ ] Happy-path (Scenario 1) работает: subscribe→ack, message.send→sent+new.
- [ ] Broadcast между клиентами (Scenario 2) работает.
- [ ] CHAT-004 для не-члена.
- [ ] CHAT-003 для voice.
- [ ] AUTH-010 до апгрейда.
- [ ] Server shutdown → close-code 1001.
- [ ] Token в логе замаскирован.
- [ ] Нет panic в логах сервера.
