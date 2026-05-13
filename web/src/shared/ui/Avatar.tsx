import { cn } from "@/shared/lib/cn";
import { uuidToColor } from "@/shared/lib/uuidToColor";

import styles from "./Avatar.module.css";

type AvatarSize = "sm" | "md" | "lg";

type AvatarProps = {
  userId: string;
  size?: AvatarSize;
};

function sizeClass(size: AvatarSize): string | undefined {
  switch (size) {
    case "sm":
      return styles.sizeSm;
    case "md":
      return styles.sizeMd;
    case "lg":
      return styles.sizeLg;
  }
}

// Декоративный аватар (D-25): цвет — детерминирован hash(userId),
// текст — первые 2 hex-символа из userId без дефисов, uppercase.
// aria-hidden, потому что имя пользователя рендерится отдельно рядом.
export function Avatar({ userId, size = "md" }: AvatarProps): JSX.Element {
  const initials = userId.replace(/-/g, "").toUpperCase().slice(0, 2);
  const background = uuidToColor(userId);

  return (
    <span
      className={cn(styles.avatar, sizeClass(size))}
      style={{ backgroundColor: background }}
      aria-hidden="true"
      data-testid="avatar"
    >
      {initials}
    </span>
  );
}
