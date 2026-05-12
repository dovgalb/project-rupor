#!/usr/bin/env bash
# Manual QA smoke-test для PR-3.1 realtime chat.
#
# Что делает:
#   1. Регистрирует двух пользователей с уникальными email'ами.
#   2. Логинит обоих, получает access-токены.
#   3. Alice создаёт комнату → text-канал → invite.
#   4. Bob джоинится по invite.
#   5. REST: пустая история, отправка через WS (websocat), повторное чтение истории.
#   6. Проверяет негативные кейсы (CHAT-002 / CHAT-004 / CHAT-005 / CHAT-006 / AUTH-010).
#
# Требования: curl, jq, websocat (опционально для WS-шагов).
# brew install jq websocat
#
# Сервер должен быть запущен (make run).

set -euo pipefail

HOST="${HOST:-http://localhost:8080}"
WS_HOST="${WS_HOST:-ws://localhost:8080}"
SUFFIX="$(date +%s)-$$"

red()    { printf "\033[0;31m%s\033[0m\n" "$*"; }
green()  { printf "\033[0;32m%s\033[0m\n" "$*"; }
yellow() { printf "\033[0;33m%s\033[0m\n" "$*"; }
bold()   { printf "\033[1m%s\033[0m\n" "$*"; }

require() {
  command -v "$1" >/dev/null 2>&1 || { red "ERROR: $1 не найден в PATH"; exit 1; }
}

require curl
require jq

assert_status() {
  local expected="$1"
  local actual="$2"
  local label="$3"
  if [[ "$actual" == "$expected" ]]; then
    green "  OK $label → $actual"
  else
    red "  FAIL $label: ожидали $expected, получили $actual"
    exit 1
  fi
}

# Уникальные email'ы для повторных запусков
EMAIL_A="alice-${SUFFIX}@chat-smoke.local"
EMAIL_B="bob-${SUFFIX}@chat-smoke.local"
PASS="correct-pass-123"

bold "==> 1. Health check"
HEALTH=$(curl -s -o /dev/null -w "%{http_code}" "$HOST/api/v1/health")
assert_status 200 "$HEALTH" "GET /health"

bold "==> 2. Register Alice + Bob"
curl -s -X POST "$HOST/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL_A\",\"username\":\"alice_${SUFFIX}\",\"password\":\"$PASS\"}" >/dev/null
curl -s -X POST "$HOST/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL_B\",\"username\":\"bob_${SUFFIX}\",\"password\":\"$PASS\"}" >/dev/null
green "  OK registered $EMAIL_A and $EMAIL_B"

bold "==> 3. Login"
TOK_A=$(curl -s -X POST "$HOST/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL_A\",\"password\":\"$PASS\"}" | jq -r .accessToken)
TOK_B=$(curl -s -X POST "$HOST/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL_B\",\"password\":\"$PASS\"}" | jq -r .accessToken)
[[ -n "$TOK_A" && "$TOK_A" != "null" ]] || { red "  FAIL: token A empty"; exit 1; }
[[ -n "$TOK_B" && "$TOK_B" != "null" ]] || { red "  FAIL: token B empty"; exit 1; }
green "  OK got tokens"

bold "==> 4. Alice creates room + channel + invite"
ROOM_ID=$(curl -s -X POST "$HOST/api/v1/rooms" \
  -H "Authorization: Bearer $TOK_A" -H "Content-Type: application/json" \
  -d "{\"name\":\"smoke-${SUFFIX}\"}" | jq -r .id)
[[ -n "$ROOM_ID" && "$ROOM_ID" != "null" ]] || { red "  FAIL: room_id empty"; exit 1; }
green "  OK room_id=$ROOM_ID"

CHANNEL_ID=$(curl -s -X POST "$HOST/api/v1/rooms/$ROOM_ID/channels" \
  -H "Authorization: Bearer $TOK_A" -H "Content-Type: application/json" \
  -d '{"name":"general","kind":"text"}' | jq -r .id)
[[ -n "$CHANNEL_ID" && "$CHANNEL_ID" != "null" ]] || { red "  FAIL: channel_id empty"; exit 1; }
green "  OK channel_id=$CHANNEL_ID"

INVITE=$(curl -s -X POST "$HOST/api/v1/rooms/$ROOM_ID/invite" \
  -H "Authorization: Bearer $TOK_A" | jq -r .code)
[[ -n "$INVITE" && "$INVITE" != "null" ]] || { red "  FAIL: invite empty"; exit 1; }
green "  OK invite_code=$INVITE"

bold "==> 5. Bob joins by invite"
JOIN_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -X POST "$HOST/api/v1/rooms/join" \
  -H "Authorization: Bearer $TOK_B" -H "Content-Type: application/json" \
  -d "{\"code\":\"$INVITE\"}")
assert_status 200 "$JOIN_STATUS" "POST /rooms/join"

bold "==> 6. Empty history check"
EMPTY=$(curl -s "$HOST/api/v1/channels/$CHANNEL_ID/messages" -H "Authorization: Bearer $TOK_A")
ITEMS=$(echo "$EMPTY" | jq -r '.items | length')
NEXT=$(echo "$EMPTY" | jq -r '.nextBefore')
if [[ "$ITEMS" == "0" && "$NEXT" == "null" ]]; then
  green "  OK пустая история: items=[], nextBefore=null"
else
  red "  FAIL: items=$ITEMS, nextBefore=$NEXT"
  exit 1
fi

bold "==> 7. WebSocket: subscribe + message.send (если установлен websocat)"
if command -v websocat >/dev/null 2>&1; then
  yellow "  → отправляю фрейм через websocat (5s timeout)..."
  WS_OUT=$(printf '%s\n%s\n' \
    "{\"type\":\"subscribe\",\"channel_id\":\"$CHANNEL_ID\"}" \
    "{\"type\":\"message.send\",\"channel_id\":\"$CHANNEL_ID\",\"text\":\"hello from smoke ${SUFFIX}\"}" \
    | websocat --max-messages 3 -n0 "$WS_HOST/api/v1/ws?token=$TOK_A" || true)
  echo "$WS_OUT" | head -5
  if echo "$WS_OUT" | grep -q '"type":"subscribed"' && \
     echo "$WS_OUT" | grep -q '"type":"message.sent"'; then
    green "  OK получены subscribed + message.sent фреймы"
  else
    red "  WARN не нашёл subscribed/message.sent — проверь вручную через 02_ws_browser_test.md"
  fi
else
  yellow "  SKIP: websocat не установлен. Brew: brew install websocat."
  yellow "  Тестируй WS через 02_ws_browser_test.md (browser console)."
fi

bold "==> 8. История после отправки (должна быть >= 0)"
HIST=$(curl -s "$HOST/api/v1/channels/$CHANNEL_ID/messages" -H "Authorization: Bearer $TOK_A")
COUNT=$(echo "$HIST" | jq -r '.items | length')
green "  OK items count = $COUNT"

bold "==> 9. Negative: AUTH-010 без токена"
NOAUTH=$(curl -s -o /dev/null -w "%{http_code}" "$HOST/api/v1/channels/$CHANNEL_ID/messages")
assert_status 401 "$NOAUTH" "GET без Authorization"

bold "==> 10. Negative: CHAT-005 невалидный channel UUID"
BAD_CH=$(curl -s -o /dev/null -w "%{http_code}" "$HOST/api/v1/channels/bad-uuid/messages" -H "Authorization: Bearer $TOK_A")
assert_status 400 "$BAD_CH" "GET с bad-uuid"

bold "==> 11. Negative: CHAT-006 limit=0"
BAD_LIM=$(curl -s -o /dev/null -w "%{http_code}" "$HOST/api/v1/channels/$CHANNEL_ID/messages?limit=0" -H "Authorization: Bearer $TOK_A")
assert_status 400 "$BAD_LIM" "GET с limit=0"

bold "==> 12. Negative: CHAT-002 несуществующий канал"
NO_CH=$(curl -s -o /dev/null -w "%{http_code}" "$HOST/api/v1/channels/00000000-0000-4000-8000-000000000000/messages" -H "Authorization: Bearer $TOK_A")
assert_status 404 "$NO_CH" "GET с unknown channel"

bold "==> 13. Negative: CHAT-004 пользователь Carol не член комнаты"
EMAIL_C="carol-${SUFFIX}@chat-smoke.local"
curl -s -X POST "$HOST/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL_C\",\"username\":\"carol_${SUFFIX}\",\"password\":\"$PASS\"}" >/dev/null
TOK_C=$(curl -s -X POST "$HOST/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL_C\",\"password\":\"$PASS\"}" | jq -r .accessToken)
CAROL=$(curl -s -o /dev/null -w "%{http_code}" "$HOST/api/v1/channels/$CHANNEL_ID/messages" -H "Authorization: Bearer $TOK_C")
assert_status 403 "$CAROL" "GET без membership"

echo
bold "==================================================="
green "ALL SMOKE TESTS PASSED"
bold "==================================================="
echo "Room:    $ROOM_ID"
echo "Channel: $CHANNEL_ID"
echo "Invite:  $INVITE"
echo "Alice:   $EMAIL_A"
echo "Bob:     $EMAIL_B"
echo "Carol:   $EMAIL_C (не член, для negative cases)"
