// Публичный API feature-слайса auth.
// Внешний код должен импортировать только отсюда.
export { useAuthStore } from "./store";
export { LoginForm } from "./components/LoginForm";
export { RegisterForm } from "./components/RegisterForm";
export { UserBadge } from "./components/UserBadge";
export type { CurrentUser, UserResponse } from "./types";
