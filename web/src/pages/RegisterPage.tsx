// Страница регистрации. После успеха — переход на /rooms (Phase 5 уточнит).

import { useNavigate } from "react-router-dom";

import { RegisterForm } from "@/features/auth";
import { ru } from "@/shared/lib/i18n/ru";

import styles from "./RegisterPage.module.css";

export function RegisterPage(): JSX.Element {
  const navigate = useNavigate();
  return (
    <div className={styles.page}>
      <h1>{ru.auth.registerTitle}</h1>
      <RegisterForm onSuccess={() => navigate("/rooms", { replace: true })} />
    </div>
  );
}
