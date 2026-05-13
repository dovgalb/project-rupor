import { useEffect, useId, useRef } from "react";
import { createPortal } from "react-dom";

import { ru } from "@/shared/lib/i18n/ru";

import styles from "./Modal.module.css";

import type { KeyboardEvent, MouseEvent, ReactNode } from "react";

type ModalProps = {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
};

// CSS-селектор фокусабельных элементов внутри dialog.
const FOCUSABLE_SELECTOR =
  'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

// Возвращает массив focusable-элементов внутри контейнера.
function getFocusableElements(container: HTMLElement): HTMLElement[] {
  return Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR));
}

// Модалка через React Portal в document.body.
// Особенности:
// - закрыт → не рендерим вообще
// - focus-trap руками (без библиотеки): Tab/Shift+Tab циклит внутри
// - Esc → onClose; клик по overlay → onClose; клик по dialog не закрывает
// - при open сохраняем previouslyFocused и возвращаем фокус при close
export function Modal({
  open,
  onClose,
  title,
  children,
}: ModalProps): JSX.Element | null {
  const dialogRef = useRef<HTMLDivElement>(null);
  const previouslyFocusedRef = useRef<HTMLElement | null>(null);
  const titleId = useId();

  useEffect(() => {
    if (!open) {
      return;
    }
    // Сохраняем фокусированный элемент до открытия.
    previouslyFocusedRef.current = document.activeElement as HTMLElement | null;

    // Ставим фокус на первый focusable внутри диалога.
    const dialog = dialogRef.current;
    if (dialog) {
      const focusables = getFocusableElements(dialog);
      const first = focusables[0];
      if (first) {
        first.focus();
      } else {
        // Если фокусабельных нет — фокус на сам контейнер (tabindex=-1).
        dialog.focus();
      }
    }

    return () => {
      // Восстанавливаем фокус на закрытии.
      const previously = previouslyFocusedRef.current;
      if (previously && typeof previously.focus === "function") {
        previously.focus();
      }
    };
  }, [open]);

  if (!open) {
    return null;
  }

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>): void => {
    if (event.key === "Escape") {
      event.stopPropagation();
      onClose();
      return;
    }
    if (event.key !== "Tab") {
      return;
    }
    // Focus-trap: циклим Tab/Shift+Tab внутри списка focusable.
    const dialog = dialogRef.current;
    if (!dialog) {
      return;
    }
    const focusables = getFocusableElements(dialog);
    if (focusables.length === 0) {
      event.preventDefault();
      return;
    }
    const first = focusables[0];
    const last = focusables[focusables.length - 1];
    if (!first || !last) {
      return;
    }
    const active = document.activeElement;

    if (event.shiftKey) {
      if (active === first || !dialog.contains(active)) {
        event.preventDefault();
        last.focus();
      }
    } else if (active === last) {
      event.preventDefault();
      first.focus();
    }
  };

  const handleOverlayMouseDown = (event: MouseEvent<HTMLDivElement>): void => {
    // Закрываем только если клик именно по overlay (не по dialog).
    if (event.target === event.currentTarget) {
      onClose();
    }
  };

  return createPortal(
    // Overlay — presentation-слой только для закрытия по клику. ARIA-роль "presentation"
    // явно сообщает screen-reader'у, что это декоративный контейнер.
    // eslint-disable-next-line jsx-a11y/no-static-element-interactions, jsx-a11y/click-events-have-key-events -- close-кнопка и Esc обеспечивают keyboard-доступность
    <div
      className={styles.overlay}
      onMouseDown={handleOverlayMouseDown}
      data-testid="modal-overlay"
      role="presentation"
    >
      {/* role=dialog + tabIndex=-1 — стандартный паттерн ARIA-модалки. onKeyDown нужен
          для focus-trap (Tab/Shift+Tab) и закрытия по Esc. Линтер ругается на onKeyDown
          у non-interactive role, но это валидное использование ARIA-pattern. */}
        {/* eslint-disable-next-line jsx-a11y/no-noninteractive-element-interactions -- ARIA dialog pattern */}
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        className={styles.dialog}
        onKeyDown={handleKeyDown}
      >
        <div className={styles.header}>
          <h2 id={titleId} className={styles.title}>
            {title}
          </h2>
          <button
            type="button"
            className={styles.close}
            onClick={onClose}
            aria-label={ru.common.close}
          >
            ×
          </button>
        </div>
        <div className={styles.body}>{children}</div>
      </div>
    </div>,
    document.body,
  );
}
