// EmptyChat: плейсхолдер пустого канала.
// Источник: docs/3_5_frontend/07-ui-contract.md (<EmptyChat />).

import { EmptyState } from "@/shared/ui";
import { ru } from "@/shared/lib/i18n/ru";

export function EmptyChat(): JSX.Element {
  return (
    <EmptyState
      title={ru.chat.emptyTitle}
      description={ru.chat.emptyDescription}
    />
  );
}
