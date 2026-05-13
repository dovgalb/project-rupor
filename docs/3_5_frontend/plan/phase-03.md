---
phase: 3
name: Shared UI primitives + utils
layer: shared
depends_on: [phase-01]
plan: ./README.md
---

# Phase 3: Shared UI primitives + utils

## Цель

Реализовать переиспользуемые UI-примитивы в `web/src/shared/ui/` и pure-утилиты в `web/src/shared/lib/`. После фазы — фичи могут использовать `<Button />`, `<Input />`, `<Modal />`, `<Toast />`, `<Avatar />`, `<Spinner />`, `<FormField />`, `<EmptyState />`, `<ErrorBoundary />`, а также `linkify`, `uuidToColor`, `formatDate`, `useToast`.

## Контекст

Phase 1 создала каркас и `theme.css`. Phase 2 параллельно (или раньше) создала shared/api. Используем `prompts/React Components.txt` (состояния, a11y, security), `prompts/TypeScript Style.txt`, ADR D-22 (тема), D-24 (свой toaster), D-25 (avatar palette).

## Файлы для создания

### UI-примитивы

#### `web/src/shared/ui/Button.tsx` (+ `.module.css` + `.test.tsx`)

```ts
type ButtonProps = {
  variant?: "primary" | "secondary" | "danger" | "ghost";
  size?: "sm" | "md";
  isLoading?: boolean;
  type?: "button" | "submit" | "reset";
} & ButtonHTMLAttributes<HTMLButtonElement>;

export function Button({ variant = "primary", size = "md", isLoading, children, ...rest }: ButtonProps);
```

- При `isLoading` — disabled + `<Spinner size="sm" />` внутри.
- `type` default `"button"`.
- CSS-классы по variant/size, цвета из `theme.css` переменных.
- Tests (~4): рендер каждого variant, disabled при isLoading, click handler не вызывается при disabled, type submit/button.

#### `web/src/shared/ui/Input.tsx` (+ `.module.css` + `.test.tsx`)

```ts
type InputProps = {
  invalid?: boolean;
} & InputHTMLAttributes<HTMLInputElement>;

export const Input = forwardRef<HTMLInputElement, InputProps>((props, ref) => { ... });
```

- `forwardRef` — для интеграции с react-hook-form.
- `aria-invalid` если `invalid`.
- Tests (~3): рендер, forwardRef работает, aria-invalid при invalid=true.

#### `web/src/shared/ui/Textarea.tsx` (+ `.module.css` + `.test.tsx`)

Аналог Input с auto-size (через `field-sizing: content` CSS или JS-логику высоты). Tests (~3).

#### `web/src/shared/ui/FormField.tsx` (+ `.test.tsx`)

```ts
type FormFieldProps = {
  label: string;
  htmlFor: string;
  error?: string;
  children: ReactNode;
};
```

- Рендерит `<label htmlFor>` + children + `<span role="alert">` если error.
- Tests (~2): рендер с error, рендер без error.

#### `web/src/shared/ui/Modal.tsx` (+ `.module.css` + `.test.tsx`)

```ts
type ModalProps = {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
};
```

- Использует `<dialog>` HTML-элемент или portal в `document.body`.
- Focus-trap: после открытия — фокус на первый focusable элемент внутри; Tab/Shift+Tab циклит внутри; Esc — `onClose`.
- `role="dialog"`, `aria-modal="true"`, `aria-labelledby={titleId}`.
- Overlay click — `onClose`.
- Tests (~5): открыт/закрыт, Esc вызывает onClose, focus trap (первый input получает фокус), aria-attributes, кнопка close в углу.

#### `web/src/shared/ui/Spinner.tsx` (+ `.module.css`)

- Props: `size?: "sm" | "md" | "lg"`.
- CSS-анимация (rotate 360deg infinite).
- `role="progressbar"`, `aria-label="Загрузка"`.

#### `web/src/shared/ui/FullScreenSpinner.tsx`

- Полноэкранный wrapper над `<Spinner size="lg" />` для guard-loading состояний.

#### `web/src/shared/ui/Toaster.tsx` + `Toast.tsx` + `useToast.ts` (+ `.module.css` + `.test.tsx`)

**Реализация (D-24, свой toaster):**

```ts
// useToast.ts — event-emitter поверх Zustand или ref-based
type ToastVariant = "info" | "success" | "warn" | "error";
type ToastItem = { id: string; variant: ToastVariant; message: string; duration: number };

// API:
toast.info("message");
toast.success("message");
toast.warn("message");
toast.error("message");

// <Toaster /> подписывается на стор, рендерит активные тосты
```

- Store: `useToastsStore` (исключение из правила «по сторе на фичу» — это `shared/ui`). Состояние `{ items: ToastItem[] }`, actions `push(item)`, `dismiss(id)`. Auto-dismiss timer внутри компонента.
- Default duration: 4s для info/success/warn, 7s для error.
- Tests (~4): push/dismiss, рендер по variant, auto-dismiss, role=status / role=alert.

#### `web/src/shared/ui/Avatar.tsx` (+ `.module.css` + `.test.tsx`)

```ts
type AvatarProps = { userId: string; size?: "sm" | "md" | "lg" };
```

- Цвет фона: `uuidToColor(userId)` (см. utils).
- Текст: первые 2 hex-символа от userId (uppercase, без дефисов).
- `aria-hidden="true"` (декоративно).
- Tests (~3): детерминированный цвет, два символа, aria-hidden.

#### `web/src/shared/ui/EmptyState.tsx` (+ `.module.css`)

```ts
type EmptyStateProps = {
  title: string;
  description?: string;
  action?: ReactNode;  // обычно <Button />
};
```

- Простой layout: иконка/emoji-placeholder + title + description + optional CTA.

#### `web/src/shared/ui/ErrorBoundary.tsx` (+ `.test.tsx`)

Классовый компонент (React требует), оборачивает `<App />`.

- `componentDidCatch` — `logger.error(error, errorInfo)`.
- Fallback UI: «Что-то пошло не так. Перезагрузите страницу.»
- Tests (~2): рендер children при ok, fallback при error.

#### `web/src/shared/ui/index.ts`

Barrel re-exports всех примитивов.

### Утилиты `shared/lib/`

#### `web/src/shared/lib/uuidToColor.ts` (+ `.test.ts`)

**D-25** — 8 фиксированных цветов с AA-контрастом.

```ts
const PALETTE = [
  "#5865f2",  // accent blue
  "#23a55a",  // success green
  "#f0b232",  // warn yellow
  "#f23f42",  // danger red
  "#8b5cf6",  // purple
  "#ec4899",  // pink
  "#06b6d4",  // cyan
  "#84cc16",  // lime
] as const;

export function uuidToColor(uuid: string): string {
  // hash := sum charcodes of uuid.replace(/-/g, "")
  // return PALETTE[hash % 8]
}
```

- Tests (~3): детерминирован для одного uuid, разные uuid могут давать разные цвета, palette indexes < 8.

#### `web/src/shared/lib/linkify.ts` (+ `.test.ts`)

```ts
export type LinkifyPart = { type: "text" | "link"; value: string; href?: string };
export function linkify(text: string): LinkifyPart[];
```

- Распознаёт `http://`, `https://`, `mailto:` URL внутри текста.
- Возвращает массив частей для рендера (text-node + `<a>`).
- НЕ преобразует `javascript:`, `data:`, `vbscript:` (возвращает как text).
- Tests (~5): plain text → один part; URL внутри текста → 3 parts; javascript: → текст; mailto:; multiple URLs.

#### `web/src/shared/lib/formatDate.ts` (+ `.test.ts`)

```ts
export function formatTime(rfc3339: string): string;  // "14:32"
export function formatDateTime(rfc3339: string): string;  // "13.05.2026 14:32"
export function formatRelative(rfc3339: string, now?: Date): string;  // "только что", "5 мин назад", ...
```

- Простая реализация через `Intl.DateTimeFormat("ru-RU", ...)`.
- Tests (~5).

#### `web/src/shared/lib/index.ts`

Barrel: re-exports `uuidToColor`, `linkify`, `formatTime`/`formatDateTime`/`formatRelative`, `logger`.

## Файлы для модификации

- `web/src/shared/lib/i18n/ru.ts` (создан в Phase 1) — добавить ключи `common`: `loading`, `cancel`, `save`, `delete`, `retry`, `close`, `copy`, `copied`, `error`, `tryAgain`, ...

## Ключевые решения

- **D-22** Тёмная тема — все компоненты используют `var(--color-*)` из `theme.css`, не хардкодят цвета.
- **D-24** Свой toaster — `<Toaster />` + `useToastsStore` + `toast.info/success/warn/error` API.
- **D-25** 8-color palette в `uuidToColor.ts`.
- a11y везде: `role`, `aria-label`, keyboard.
- Никаких `dangerouslySetInnerHTML` (см. `prompts/React Components.txt`).

## Verification

- [ ] `npm --prefix web run typecheck` — 0 ошибок.
- [ ] `npm --prefix web run lint` — 0 ошибок.
- [ ] `npm --prefix web run test -- --run` — все ~35 тестов shared/ui + shared/lib зелёные.
- [ ] Все компоненты со `state` имеют тесты на каждое состояние (`isLoading`, `invalid`, `open/closed`, `error/ok`).
- [ ] Modal: focus trap проверен в тесте (первый focusable получает фокус после open).
- [ ] Toaster: auto-dismiss проверен через `vi.useFakeTimers()`.
- [ ] linkify не пропускает `javascript:` (security test).
- [ ] uuidToColor детерминирован (test) и все индексы < 8.
- [ ] Никаких `any`, никаких `dangerouslySetInnerHTML`.
- [ ] `<Button>` использует `<button>`, не `<div>`.
