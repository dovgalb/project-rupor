// LoginForm: react-hook-form + zod. Server-error → form.setError.
// Источник: docs/3_5_frontend/07-ui-contract.md, plan/phase-04.md.

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { Link } from "react-router-dom";

import { loginSchema } from "@/features/auth/components/LoginForm.schema";
import { useAuthStore } from "@/features/auth/store";
import { mapAuthErrorToMessage } from "@/features/auth/store/mapErrors";
import { Button } from "@/shared/ui/Button";
import { FormField } from "@/shared/ui/FormField";
import { Input } from "@/shared/ui/Input";
import { ru } from "@/shared/lib/i18n/ru";

import styles from "./LoginForm.module.css";

import type { LoginValues } from "@/features/auth/components/LoginForm.schema";

type LoginFormProps = {
  onSuccess: () => void;
};

export function LoginForm({ onSuccess }: LoginFormProps): JSX.Element {
  const form = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: "", password: "" },
  });

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = form;

  async function onSubmit(values: LoginValues): Promise<void> {
    const ok = await useAuthStore.getState().login(values);
    if (!ok) {
      const code = useAuthStore.getState().lastErrorCode;
      const message =
        code === "AUTH-006"
          ? ru.auth.invalidCredentials
          : mapAuthErrorToMessage(code ?? "INTERNAL");
      setError("root.serverError", { message });
      return;
    }
    onSuccess();
  }

  const serverErrorMessage = errors.root?.serverError?.message;

  return (
    <form
      className={styles.form}
      aria-label={ru.auth.loginFormLabel}
      onSubmit={handleSubmit(onSubmit)}
      noValidate
    >
      <FormField
        label={ru.auth.email}
        htmlFor="login-email"
        {...(errors.email?.message !== undefined ? { error: errors.email.message } : {})}
      >
        <Input
          id="login-email"
          type="email"
          autoComplete="email"
          invalid={errors.email !== undefined}
          {...register("email")}
        />
      </FormField>

      <FormField
        label={ru.auth.password}
        htmlFor="login-password"
        {...(errors.password?.message !== undefined
          ? { error: errors.password.message }
          : {})}
      >
        <Input
          id="login-password"
          type="password"
          autoComplete="current-password"
          invalid={errors.password !== undefined}
          {...register("password")}
        />
      </FormField>

      {serverErrorMessage !== undefined ? (
        <div role="alert" className={styles.serverError}>
          {serverErrorMessage}
        </div>
      ) : null}

      <Button type="submit" isLoading={isSubmitting}>
        {ru.auth.loginBtn}
      </Button>

      <div className={styles.footer}>
        <Link to="/register" className={styles.link}>
          {ru.auth.toRegister}
        </Link>
      </div>
    </form>
  );
}
