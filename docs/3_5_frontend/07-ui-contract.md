---
parent: ./README.md
feature: 3_5-frontend
---

# 3.5 Frontend — UI Contract

Спецификация всех новых UI-компонентов PR-3.5: назначение, props, состояния, интеракции, a11y, граничные случаи. ASCII-мокапы — для фиксации layout, не для пиксельной точности.

## Общие требования

- Все интерактивные элементы — кнопки/инпуты/линки с доступными именами (`aria-label` или видимый text).
- Keyboard navigation: Tab по полям/кнопкам в логическом порядке; Enter — primary action; Esc — закрытие модалок.
- Loading-состояния показываем спиннером (общий `<Spinner />` в `shared/ui/`).
- Error-состояния: inline под полем формы (field error), form-level (над формой), или toast (для глобальных ошибок).
- Длинный текст переноситься / труcer'ится с `text-overflow: ellipsis` (CSS Modules).
- Цветовой контраст ≥ AA по WCAG.
- Запрет `dangerouslySetInnerHTML` (см. `prompts/React Components.txt`).

## Компоненты по фичам

---

## features/auth

### `<LoginForm />`

**Назначение:** форма логина. Используется на `LoginPage`.

**Props:**
| Имя | Тип | Обязат. | Описание |
|---|---|---|---|
| `onSuccess` | `() => void` | да | callback после успешного логина (обычно `navigate("/rooms")`) |

**Состояния:**

| State | Условие | UI |
|---|---|---|
| idle | дефолт | пустая форма, кнопка enabled |
| submitting | `isSubmitting` true | поля disabled, кнопка показывает `<Spinner />` |
| error | server-error из store | form-level error message над формой |

**Layout:**

```
+---------------------------------------+
| Войти в Rupor                         |
+---------------------------------------+
| [Email                              ] |
| [error если есть                    ] |
| [Пароль                             ] |
| [error если есть                    ] |
| (form-level error "Неверный...")      |
| [           Войти           ]         |
| Нет аккаунта? → Регистрация            |
+---------------------------------------+
```

**Интеракции:**
- Tab по `email` → `password` → submit-кнопке → линку «Регистрация».
- Enter в любом поле — submit формы.
- Esc — без действия (модалкой не является).
- Submit вызывает `useAuthStore.login()`; на ok — `props.onSuccess()`.

**a11y:**
- `<form>` с `aria-label="Форма входа"`.
- Каждый `<input>` имеет `<label htmlFor>`.
- Кнопка submit — `<button type="submit">`.
- `aria-invalid` на полях с ошибкой.
- `role="alert"` на form-level error.

**Граничные случаи:**
- Длинный email (>50 символов) — поле скроллится горизонтально.
- Двойной submit (быстрый Enter Enter) — second Enter игнорируется через `isSubmitting`.

### `<RegisterForm />`

Аналогично `<LoginForm />`, но с дополнительным полем `username`. Схема валидации zod: `email` + `username` (alnum/_/- 3..32) + `password` (8..72).

**Состояния:** те же.

**Layout:**

```
+---------------------------------------+
| Создать аккаунт                       |
+---------------------------------------+
| [Email                              ] |
| [Имя пользователя                   ] |
| [Пароль                             ] |
| (form-level error если есть)          |
| [        Создать аккаунт        ]     |
| Уже есть аккаунт? → Войти              |
+---------------------------------------+
```

**Field errors mapping:**
- `AUTH-001` → поле `email`.
- `AUTH-002` → поле `username`.
- `AUTH-003` → поле `password`.
- `AUTH-004` → поле `email` («Email уже занят»).
- `AUTH-005` → поле `username` («Имя пользователя уже занято»).

### `<UserBadge />`

**Назначение:** отображение текущего пользователя в TopBar.

**Props:**
| Имя | Тип | Обязат. | Описание |
|---|---|---|---|
| `user` | `CurrentUser` | да | данные `/auth/me` |
| `onLogout` | `() => void` | да | callback на «Выйти» |

**Состояния:** только success (компонент рендерится только когда `currentUser` есть).

**Layout:**

```
+----------------------------------+
| (avatar) username        [Выйти] |
+----------------------------------+
```

`avatar` — генерируемый по `user.id` (см. `<Avatar />` ниже).

**Интеракции:**
- Click на «Выйти» → `onLogout()`.
- Hover на username — tooltip с полным email.

**a11y:** кнопка «Выйти» — `<button>`, `aria-label="Выйти из аккаунта"`.

---

## features/rooms

### `<RoomList />`

**Назначение:** sidebar с моими комнатами + кнопки создания/вступления.

**Props:**
| Имя | Тип | Обязат. | Описание |
|---|---|---|---|
| `activeRoomId` | `string \| null` | да | подсвечивается |

**Состояния:**
| State | Условие | UI |
|---|---|---|
| loading | `status==="loading"` | список плейсхолдеров-скелетонов |
| empty | `rooms.length===0` | EmptyState с CTA «Создать» / «Войти по коду» |
| error | `status==="error"` | inline error + кнопка «Повторить» |
| ready | данные есть | список комнат |

**Layout:**

```
+---------------------+
| Мои комнаты    [+ ] |  <- "+" открывает меню {Create, Join}
+---------------------+
| (active) #general   |  <- комната + первая буква названия как «лого»
|   #random           |
|   #backend          |
| ...                 |
+---------------------+
```

**Интеракции:**
- Click по room → `selectRoom(roomId)` + `navigate("/rooms/:roomId")`.
- Click `+` → popover с {Create room, Join by code}.

**a11y:** список как `<nav><ul role="list">`. Активный item — `aria-current="page"`.

### `<RoomListItem />`

**Назначение:** одна комната в сайдбаре.

**Props:** `{ room: RoomWithRole; active: boolean; onClick: () => void }`.

**Состояния:** только два — active / inactive.

**a11y:** `<button>` или `<Link>` (предпочесть Link для роутинга).

### `<CreateRoomModal />`

**Назначение:** модалка создания комнаты.

**Props:** `{ open: boolean; onClose: () => void; onCreated: (room: RoomWithRole) => void }`.

**Состояния:** idle / submitting / error.

**Layout:**

```
+----------------------------------+
| Создать комнату            [X]   |
+----------------------------------+
| [Название (1..64)              ] |
| (error)                          |
| [   Отмена   ]    [  Создать  ]  |
+----------------------------------+
```

**Интеракции:** Esc — close. Enter в поле — submit.

**a11y:** `role="dialog"`, `aria-modal="true"`, focus trap внутри.

### `<DeleteRoomConfirm />`

**Назначение:** подтверждение удаления комнаты.

**Props:** `{ open: boolean; roomName: string; onConfirm: () => Promise<void>; onCancel: () => void }`.

**Layout:**

```
+----------------------------------+
| Удалить комнату «{name}»?  [X]   |
+----------------------------------+
| Это действие нельзя отменить.    |
| Все каналы и сообщения будут     |
| удалены безвозвратно.            |
|                                  |
| [  Отмена  ]  [  Удалить  ]      |
+----------------------------------+
```

Кнопка «Удалить» — `aria-label="Удалить комнату {name}"`, цвет danger.

### `<InviteCodeModal />`

**Назначение:** показать инвайт-код комнаты + кнопка «Сгенерировать новый».

**Props:** `{ open: boolean; roomId: string; onClose: () => void }`.

**Состояния:**

| State | UI |
|---|---|
| no-code-yet | placeholder + кнопка «Сгенерировать код» |
| generating | спиннер на кнопке |
| has-code | код показан + кнопка «Скопировать» + кнопка «Сгенерировать новый» |
| error | toast (см. error mapping в `02-behavior.md`) |

**Layout (has-code):**

```
+----------------------------------+
| Пригласить в комнату       [X]   |
+----------------------------------+
| Покажи этот код другу:           |
|                                  |
|     ABCD1234     [Скопировать]   |
|                                  |
| Создание нового кода отзовёт     |
| предыдущий.                      |
|                                  |
| [ Сгенерировать новый код ]      |
+----------------------------------+
```

**Граничный случай:** `Copy` через `navigator.clipboard.writeText` — fallback на `document.execCommand("copy")` для старых браузеров. Toast «Скопировано».

### `<JoinByCodeModal />`

**Props:** `{ open: boolean; onClose: () => void; onJoined: (room: Room) => void }`.

**Layout:**

```
+----------------------------------+
| Войти по коду              [X]   |
+----------------------------------+
| Введи код приглашения (8 симв.)  |
| [ _ _ _ _ _ _ _ _              ] |
| (error)                          |
| [   Отмена   ]    [ Войти ]      |
+----------------------------------+
```

Поле может быть 8 отдельных боксов (cinematic) или одно с `maxlength=8`. Для MVP — одно поле с автоформатированием (upper-case, only Crockford-алфавит).

**Validation (zod):**
```ts
z.string().length(8).regex(/^[0-9A-HJKMNP-TV-Z]{8}$/)  // Crockford
```

### `<MembersList />`

**Назначение:** список участников активной комнаты в сайдбаре.

**Props:** `{ roomId: string }`.

**Состояния:** loading / empty (теоретически невозможно) / ready / error.

**Layout:**

```
+---------------------+
| Участники (N)       |
+---------------------+
| (avatar) abc123 [owner] |
| (avatar) def456 [admin] |
| (avatar) g7h8i9         |
| ...                 |
+---------------------+
```

Имя — UUID-placeholder (первые 6 символов user_id). Для текущего пользователя — реальный username.

### `<MemberListItem />`

**Props:** `{ member: Member; isMe: boolean; meUsername?: string }`.

Если `isMe` — отображаем `meUsername` вместо UUID-placeholder.

**a11y:** role badge — отдельный `<span>` с `aria-label="роль: владелец"`.

### `<Avatar />`

**Назначение:** круглый avatar по `userId`.

**Props:** `{ userId: string; size?: "sm" | "md" | "lg" }`.

**Логика:**
- Цвет фона — детерминированный по hash(userId), из палитры 8 цветов (фиксированы в `shared/lib/uuidToColor.ts`).
- Текст внутри — первые 2 символа от UUID после маски (uppercase).

**Layout:**

```
   ___
  /   \
 | AB  |     bg-color по hash(userId), text white
  \___/
```

**a11y:** `aria-hidden="true"` (декоративный), имя пользователя рядом отдельным текстом.

---

## features/channels

### `<ChannelList />`

**Назначение:** список каналов активной комнаты (text + voice).

**Props:** `{ roomId: string; activeChannelId: string | null; canManage: boolean }`.

`canManage` — true для admin/owner.

**Состояния:** loading / empty / ready / error.

**Layout:**

```
+----------------------------+
| Каналы            [+]      |  <- + только если canManage
+----------------------------+
| Text channels              |
| # (active) general         |
|   random                   |
| Voice channels (coming)    |
|   [voice]general (disabled)|
| ...                        |
+----------------------------+
```

Voice-секция — disabled, fontStyle italic, tooltip «Голосовые каналы появятся позже».

### `<ChannelListItem />`

**Props:** `{ channel: Channel; active: boolean; disabled?: boolean; onSelect: () => void }`.

Voice — `disabled` props=true, `aria-disabled="true"`, click игнорируется.

### `<CreateChannelModal />`

**Props:** `{ open: boolean; roomId: string; onClose: () => void; onCreated: (channel: Channel) => void }`.

**Layout:**

```
+----------------------------------+
| Новый канал                [X]   |
+----------------------------------+
| [Название (1..64)              ] |
| Тип:  (•) Text   ( ) Voice       |
| (error)                          |
| [   Отмена   ]    [ Создать ]    |
+----------------------------------+
```

**Note:** если `kind === "voice"` — после создания UI показывает toast «Канал создан, но голосовая часть появится позже».

### `<DeleteChannelConfirm />`

Аналог `<DeleteRoomConfirm />` для канала. Доступен только admin/owner.

---

## features/chat

### `<ChatPanel />`

**Назначение:** корневой компонент центральной панели. Собирает все sub-компоненты.

**Props:** `{ channelId: string }`.

**Layout:**

```
+--------------------------------------+
| Channel header: #general    (info)   |  <- topbar канала
+--------------------------------------+
| [ConnectionStatusBanner если нужно]  |
+--------------------------------------+
| MessageList (scrollable, fills space)|
|                                      |
|                                      |
|                                      |
+--------------------------------------+
| MessageComposer (sticky bottom)      |
+--------------------------------------+
```

### `<MessageList />`

**Назначение:** список сообщений + auto-scroll + infinite-scroll вверх.

**Props:** `{ channelId: string }`.

**Состояния:**

| State | Условие | UI |
|---|---|---|
| loading | `loadingHistoryByChannel[cid]===true` | спиннер по центру |
| empty | `messages.length===0` | `<EmptyChat />` |
| ready | `messages.length>0` | список |
| loading-more | `loadingMoreHistoryByChannel[cid]` | спиннер вверху |
| end-of-history | `nextBeforeByChannel[cid]===null` | подпись «Это начало канала» |

**Логика прокрутки:**
- На новое сообщение (`message.new` / `message.sent`) — авто-скролл к низу, если пользователь УЖЕ был у низа (порог 100px от низа). Иначе — показать pill `↓ Новые сообщения (N)`, click — скролл к низу.
- На scroll к верху (порог 200px) — `loadMoreHistory(cid)` если `nextBefore` не null.

**a11y:** `role="log"`, `aria-live="polite"` (новые сообщения зачитываются screen reader).

### `<MessageItem />`

**Props:** `{ message: Message; isAuthor: boolean; authorPlaceholder: string }`.

`authorPlaceholder` = первые 6 символов `authorId` (или username, если автор = я и есть `currentUser`).

**Layout:**

```
(avatar) authorPlaceholder · 14:32
text...
text...
[значок статуса для своих pending/failed]
```

**Статусы (для своих сообщений):**
- `pending` — серый, в углу маленькая иконка «часы».
- `committed` — обычно (без иконки).
- `failed` — красная иконка ⚠, кнопка «Повторить» рядом.

**Граничные случаи:**
- Длинный текст с переносами — рендерится как text-node, `white-space: pre-wrap`.
- URL внутри текста — преобразуется в кликабельный `<a>` через `shared/lib/linkify.ts` (валидация протоколов http/https/mailto; `target="_blank" rel="noopener noreferrer"`).
- НЕ рендерить HTML.

### `<MessageComposer />`

**Назначение:** textarea с кнопкой отправки.

**Props:** `{ channelId: string; disabled?: boolean }`.

`disabled` — если WS не open или нет доступа.

**Layout:**

```
+--------------------------------------+
| [Напиши сообщение...           ] [→] |
| 0/4000  (счётчик показывать >3000)   |
+--------------------------------------+
```

**Интеракции:**
- Enter — submit (если text не пустой).
- Shift+Enter — новая строка.
- Текстовое поле autosize по содержимому (max 6 строк, дальше scroll внутри textarea).

**Состояния:**
- idle / sending / error (если ws-error пришёл — inline под composer).
- disabled (wsStatus не open) — поле тусклое, placeholder «Соединение недоступно».

**a11y:** `aria-label="Сообщение в канал {channelName}"`. Кнопка отправки — `aria-label="Отправить"`.

### `<ConnectionStatusBanner />`

**Назначение:** баннер при потере WS-связи.

**Props:** `{}` — читает `wsStatus` из стора.

**Layout (reconnecting):**

```
+--------------------------------------+
| ⏳ Переподключение…                  |
+--------------------------------------+
```

Жёлтый фон. Исчезает при `wsStatus === "open"`.

**a11y:** `role="status"`, `aria-live="polite"`.

### `<EmptyChat />`

```
+--------------------------------------+
|                                      |
|         Здесь пока пусто             |
|         Напиши первое сообщение      |
|                                      |
+--------------------------------------+
```

---

## Каркас приложения

### `<AppShell />`

**Назначение:** трёхпанельный layout, обрамляющий все авторизованные экраны.

**Props:** `{ children: ReactNode }`.

**Layout:**

```
+----------+--------------------------+
|  Sidebar | TopBar (UserBadge)       |
|          +--------------------------+
| RoomList | ChannelList | ChatPanel  |
|          | MembersList |            |
|          |             |            |
+----------+-------------+------------+
```

Сайдбар коллапсируемый на mobile (после MVP).

### `<TopBar />`

**Назначение:** верхняя панель с `<UserBadge />` справа + опционально титул текущей комнаты слева.

**Props:** `{}`.

### `<RoomScopedSidebar />`

**Назначение:** вложенный sidebar, виден только когда выбрана комната (`:roomId` в URL). Содержит `<ChannelList />` сверху и `<MembersList />` снизу.

**Props:** `{}` — берёт `roomId` из `useParams`.

**Состояния:** скрыт целиком если `roomId` нет; иначе показывает оба child-компонента.

**Layout:**

```
+----------------+
| ChannelList    |
| ...            |
+----------------+
| MembersList    |
| ...            |
+----------------+
```

Скролл — независимый внутри каждого блока.

### `<NotFoundPage />`

**Назначение:** fallback для неизвестных роутов.

**Layout:**

```
+----------------------------------+
| 404 — Страница не найдена         |
| [На главную]                     |
+----------------------------------+
```

---

## Toaster

### `<Toaster />` (global)

**Назначение:** глобальный контейнер для toast-уведомлений. Подключается в `app/App.tsx`.

**API (через event-emitter или контекст):**

```ts
toast.info("сообщение");
toast.success("...");
toast.error("...");
toast.warn("...");
```

**Layout:** правый нижний угол, stack снизу вверх, авто-исчезновение 4s (для error — 7s).

**a11y:** `role="status"` / `role="alert"` для error.

**Тип реализации:** собственный (см. `03-decisions.md` open question — голос за «свой toaster», финал — в плане кода).

---

## Shared UI примитивы

`web/src/shared/ui/`:

| Компонент | Назначение |
|---|---|
| `<Button />` | базовая кнопка, варианты primary/secondary/danger/ghost, size sm/md |
| `<Input />` | text input + `<label>` integration |
| `<Textarea />` | autosize textarea |
| `<FormField />` | wrapper: label + input + error message |
| `<Modal />` | dialog с overlay, focus trap, Esc-close |
| `<Spinner />` | крутилка, size sm/md/lg |
| `<Toast />` | элемент toast'а |
| `<Avatar />` | (описан выше) |
| `<EmptyState />` | placeholder с иллюстрацией/иконкой + title + description + CTA |
| `<ErrorBoundary />` | React error boundary, fallback UI |

Все примитивы — pure-presentational, без бизнес-логики и Zustand-зависимостей.

---

## Тема и CSS

CSS Modules. Файлы рядом с компонентами: `LoginForm.tsx` + `LoginForm.module.css`.

**Глобальные CSS-переменные** — в `web/src/app/styles/theme.css`:

```css
:root {
  /* Темная тема (Discord-like) — default */
  --color-bg-primary:   #1e1f22;
  --color-bg-secondary: #2b2d31;
  --color-bg-tertiary:  #313338;
  --color-text-primary: #f2f3f5;
  --color-text-muted:   #b5bac1;
  --color-accent:       #5865f2;
  --color-danger:       #f23f42;
  --color-success:      #23a55a;
  --color-warning:      #f0b232;
  --radius-sm: 4px;
  --radius-md: 8px;
  --font-base: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
}
```

Светлая тема — после MVP. Файл переменных позволит легко переключить.

---

## Стейт-управление в формах: react-hook-form + zod

Конкретный пример — см. `05-state-model.md`. Каждая форма имеет:
- `*.schema.ts` — zod-схема + `type *Values = z.infer<typeof schema>`
- `*.tsx` — компонент с `useForm({ resolver: zodResolver(schema) })`
- Server-error mapping — в `onSubmit` через `form.setError("fieldName", { message })`.

---

## Проверка соответствия `prompts/React Components.txt`

| Требование стандарта | Применение в этой фиче |
|---|---|
| Все 5 состояний (idle/loading/empty/error/success) | Перечислены для каждого компонента, имеющего state |
| `aria-label` на интерактивных без видимого имени | Кнопки `[X]` close, иконки «копировать», отправка |
| Кнопки — `<button>`, не `<div onClick>` | Везде `<button>` или `<Link>` |
| Никаких `dangerouslySetInnerHTML` | Текст сообщений как text-node, ссылки через linkify с валидацией протокола |
| Keyboard: Enter/Esc/Tab | Описано в каждом интерактивном компоненте |
| Стабильные `key` в списках | `room.id`, `channel.id`, `message.id` (или `tempId` для pending) |
