# Manual QA: WebSocket-чат через browser console

**Prerequisite:** сервер запущен (`make run`), есть валидный access token и channel_id из 00_flow.http.

## ⚠️ Важно про CSP

Не открывай DevTools на сайте с жёсткой Content-Security-Policy (reddit, github, gmail и т.п.) — `connect-src` запретит `ws://localhost`. Используй вариант БЕЗ CSP:

- **Готовая страничка:** открой `manual_qa/3_1_realtime_chat/ws_test.html` — там UI с кнопками `Connect / Subscribe / message.send`.
- **`about:blank`** в новой вкладке → DevTools → Console → вставляй js-сниппеты ниже.
- **`websocat` CLI** — без браузера, см. ниже §"Альтернатива: websocat".

## ⚠️ Важно про Origin (CORS upgrade-check)

`pkg/websocket.Upgrade` ставит `OriginPatterns` из `CORS_ALLOWED_ORIGINS` (`.env`). Если Origin твоей страницы НЕ в whitelist — WS-handshake вернёт **403** ("request Origin ... is not authorized for Host"), браузер закроет с code 1006.

Подводный камень — если открыть `ws_test.html` через **встроенный HTTP-сервер GoLand** (`localhost:63342`) или **двойной клик** (`file://` → Origin null) — Origin не совпадёт с `http://localhost:5173` из default-конфига.

**Workaround:** подними static-server на 5173 в рабочем каталоге проекта и открой через него:

```bash
python3 -m http.server 5173
# затем в браузере:
# http://localhost:5173/manual_qa/3_1_realtime_chat/ws_test.html
```

Альтернатива — добавь свой dev-origin в `.env`:

```
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:63342
```

и перезапусти сервер.

## Сценарий 1: connect + subscribe + message.send (happy path)

1. Открой Chrome DevTools → Console **на about:blank** (или используй `ws_test.html`).
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

## Альтернатива: websocat (CLI без браузера)

```bash
brew install websocat
TOKEN="<access token>"
CHANNEL="<channel uuid>"

# интерактивный режим: печатаешь JSON-фрейм Enter'ом, видишь входящие
websocat "ws://localhost:8080/api/v1/ws?token=$TOKEN"

# one-shot subscribe + message.send из stdin:
printf '%s\n%s\n' \
  '{"type":"subscribe","channel_id":"'$CHANNEL'"}' \
  '{"type":"message.send","channel_id":"'$CHANNEL'","text":"hello via websocat"}' \
  | websocat --max-messages 3 -n0 "ws://localhost:8080/api/v1/ws?token=$TOKEN"
```

`manual_qa/3_1_realtime_chat/run_smoke.sh` использует именно websocat — запусти его для одной автоматической проверки REST+WS.

## Post-test чек-лист
- [ ] Happy-path (Scenario 1) работает: subscribe→ack, message.send→sent+new.
- [ ] Broadcast между клиентами (Scenario 2) работает.
- [ ] CHAT-004 для не-члена.
- [ ] CHAT-003 для voice.
- [ ] AUTH-010 до апгрейда.
- [ ] Server shutdown → close-code 1001.
- [ ] Token в логе замаскирован.
- [ ] Нет panic в логах сервера.
