// Форматирование дат под русскую локаль (D-21).
// Принимаем RFC3339-строку (формат бэка для timestamps), возвращаем строку для UI.

const TIME_FORMATTER = new Intl.DateTimeFormat("ru-RU", {
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});

const DATE_TIME_FORMATTER = new Intl.DateTimeFormat("ru-RU", {
  day: "2-digit",
  month: "2-digit",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});

// "14:32" (24-часовой формат).
export function formatTime(rfc3339: string): string {
  const date = new Date(rfc3339);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  return TIME_FORMATTER.format(date);
}

// "13.05.2026, 14:32" (Intl сам ставит запятую в ru-RU).
export function formatDateTime(rfc3339: string): string {
  const date = new Date(rfc3339);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  return DATE_TIME_FORMATTER.format(date);
}

// Относительное время с порогами:
// <60s → "только что"
// 1..59m → "N мин назад"
// 1..23h → "N ч назад"
// 1..6 дн → "N дн назад"
// 7+ дн → полная дата через formatDateTime
export function formatRelative(rfc3339: string, now: Date = new Date()): string {
  const date = new Date(rfc3339);
  if (Number.isNaN(date.getTime())) {
    return "";
  }

  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 60) {
    return "только что";
  }

  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) {
    return `${diffMin} мин назад`;
  }

  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) {
    return `${diffHour} ч назад`;
  }

  const diffDay = Math.floor(diffHour / 24);
  if (diffDay < 7) {
    return `${diffDay} дн назад`;
  }

  return formatDateTime(rfc3339);
}
