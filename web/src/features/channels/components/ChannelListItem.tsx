// ChannelListItem: один канал в списке.
// Источник: docs/3_5_frontend/07-ui-contract.md (<ChannelListItem />).
// D-12: voice — disabled, click игнорируется, aria-disabled, tooltip.

import { cn } from "@/shared/lib/cn";
import { ru } from "@/shared/lib/i18n/ru";

import styles from "./ChannelListItem.module.css";

import type { Channel } from "@/features/channels/types";

type ChannelListItemProps = {
  channel: Channel;
  active: boolean;
  disabled?: boolean;
  onSelect: () => void;
};

// Префиксы каналов: # для text, [voice] для voice.
// CLAUDE.md запрещает эмодзи в коде — текстовые префиксы.
const PREFIX_TEXT = "#";
const PREFIX_VOICE = "[voice]";

export function ChannelListItem({
  channel,
  active,
  disabled,
  onSelect,
}: ChannelListItemProps): JSX.Element {
  const isDisabled = disabled === true;
  const prefix = channel.kind === "voice" ? PREFIX_VOICE : PREFIX_TEXT;

  function handleClick(): void {
    if (isDisabled) {
      return;
    }
    onSelect();
  }

  return (
    <li className={styles.item}>
      <button
        type="button"
        className={cn(
          styles.row,
          active && styles.active,
          isDisabled && styles.disabled,
        )}
        onClick={handleClick}
        aria-disabled={isDisabled ? true : undefined}
        aria-current={active ? "page" : undefined}
        title={isDisabled ? ru.channels.voiceComingSoonTooltip : undefined}
      >
        <span className={styles.prefix} aria-hidden="true">
          {prefix}
        </span>
        <span className={styles.name}>{channel.name}</span>
      </button>
    </li>
  );
}
