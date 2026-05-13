// useAuthStore: Zustand-store домена auth с persist (только currentUser).
// Источник: docs/3_5_frontend/05-state-model.md (useAuthStore).
// Решения: D-04 (localStorage), D-18 (один store на фичу), D-19 (persist только currentUser).

import { create } from "zustand";
import { persist } from "zustand/middleware";

import * as authApi from "@/features/auth/api/http";
import { mapAuthErrorToMessage } from "@/features/auth/store/mapErrors";
import { tokenStorage } from "@/shared/api/token-storage";
import { registerLogoutListener } from "@/shared/lib/logoutFlow";

import type {
  CurrentUser,
  LoginRequest,
  RegisterRequest,
} from "@/features/auth/types";

export type AuthStatus = "idle" | "loading" | "authenticated" | "error";

type AuthState = {
  status: AuthStatus;
  currentUser: CurrentUser | null;
  error: string | null;
  // Код последней доменной ошибки — нужен формам, чтобы смаппить на конкретное поле.
  lastErrorCode: string | null;
};

type AuthActions = {
  register: (req: RegisterRequest) => Promise<boolean>;
  login: (req: LoginRequest) => Promise<boolean>;
  loadMe: () => Promise<boolean>;
  logout: () => void;
  clear: () => void;
};

const initialState: AuthState = {
  status: "idle",
  currentUser: null,
  error: null,
  lastErrorCode: null,
};

export const useAuthStore = create<AuthState & AuthActions>()(
  persist(
    (set, get) => ({
      ...initialState,

      async register(req) {
        set({ status: "loading", error: null, lastErrorCode: null });
        const r = await authApi.register(req);
        if (!r.ok) {
          set({
            status: "error",
            error: mapAuthErrorToMessage(r.error.code),
            lastErrorCode: r.error.code,
          });
          return false;
        }
        // Auto-login сразу после успешной регистрации (UC-1).
        return get().login({ email: req.email, password: req.password });
      },

      async login(req) {
        set({ status: "loading", error: null, lastErrorCode: null });
        const r = await authApi.login(req);
        if (!r.ok) {
          set({
            status: "error",
            error: mapAuthErrorToMessage(r.error.code),
            lastErrorCode: r.error.code,
          });
          return false;
        }
        tokenStorage.set({
          access: r.data.accessToken,
          refresh: r.data.refreshToken,
          accessExpiresAt: r.data.accessExpiresAt,
          refreshExpiresAt: r.data.refreshExpiresAt,
        });
        await get().loadMe();
        return true;
      },

      async loadMe() {
        const r = await authApi.me();
        if (r.ok) {
          set({
            status: "authenticated",
            currentUser: r.data,
            error: null,
            lastErrorCode: null,
          });
          return true;
        }
        // На fail просто откатываемся в idle: logout-flow при AUTH-007..010
        // запускается автоматически в apiFetch (см. shared/api/fetch.ts).
        set({ status: "idle", currentUser: null });
        return false;
      },

      logout() {
        tokenStorage.clear();
        // Каскадная очистка остальных сторов будет дополнена в Phase 6+ через logoutFlow.
        get().clear();
      },

      clear() {
        set({ ...initialState });
      },
    }),
    {
      name: "rupor.auth",
      version: 1,
      // Персистим только currentUser. status/error/lastErrorCode после reload
      // пересчитываются через loadMe (см. UC-16).
      partialize: (state) => ({ currentUser: state.currentUser }),
    },
  ),
);

// Регистрируем clear() как listener для каскадного logoutFlow.
// Это разрывает циклический импорт shared/lib/logoutFlow → features/auth/store
// (см. правило 5 в 01-architecture.md).
registerLogoutListener(() => {
  useAuthStore.getState().clear();
});
