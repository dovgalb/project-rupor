// NotFoundPage: fallback для неизвестных роутов.
// Источник: docs/3_5_frontend/07-ui-contract.md (NotFoundPage).

import { Link } from "react-router-dom";

import { EmptyState } from "@/shared/ui";
import { ru } from "@/shared/lib/i18n/ru";

export function NotFoundPage(): JSX.Element {
  return (
    <div style={{ padding: 24 }}>
      <EmptyState
        title={ru.common.notFound}
        action={<Link to="/rooms">{ru.nav.toMain}</Link>}
      />
    </div>
  );
}
