// Разбор текста на текстовые куски и безопасные ссылки.
// Распознаём только http://, https://, mailto: — это R-01 (XSS-митигация).
// Никаких javascript:, data:, vbscript:, file:, blob: и т.п.

export type LinkifyPart =
  | { type: "text"; value: string }
  | { type: "link"; value: string; href: string };

// Якорим протокол явно: \b предотвращает совпадение с "xhttp://...".
// Маркер ссылки: пробельные/конец строки или ограничители.
// Используем нежадный matchAll и строим parts по индексам совпадений.
const URL_REGEX =
  // eslint-disable-next-line no-useless-escape -- символы оставлены для читаемости класса
  /\b(https?:\/\/[^\s<>"'`]+|mailto:[^\s<>"'`]+)/gi;

export function linkify(text: string): LinkifyPart[] {
  if (text.length === 0) {
    return [];
  }

  const parts: LinkifyPart[] = [];
  let cursor = 0;

  for (const match of text.matchAll(URL_REGEX)) {
    const url = match[0];
    const start = match.index ?? 0;

    if (start > cursor) {
      parts.push({ type: "text", value: text.slice(cursor, start) });
    }

    // Дополнительная защита: ещё раз проверяем префикс на месте.
    if (isSafeUrl(url)) {
      parts.push({ type: "link", value: url, href: url });
    } else {
      parts.push({ type: "text", value: url });
    }

    cursor = start + url.length;
  }

  if (cursor < text.length) {
    parts.push({ type: "text", value: text.slice(cursor) });
  }

  // Если URL не найдено — возвращаем весь текст как text-part.
  if (parts.length === 0) {
    return [{ type: "text", value: text }];
  }

  return parts;
}

function isSafeUrl(url: string): boolean {
  const lower = url.toLowerCase();
  return (
    lower.startsWith("http://") ||
    lower.startsWith("https://") ||
    lower.startsWith("mailto:")
  );
}
