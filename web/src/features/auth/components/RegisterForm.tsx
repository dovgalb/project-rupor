// RegisterForm: email/username/password.
// Server-error коды маппятся на конкретные поля (AUTH-001/004 → email и т.д.).

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { Link } from "react-router-dom";

import { registerSchema } from "@/features/auth/components/RegisterForm.schema";
import { useAuthStore } from "@/features/auth/store";
import { mapAuthErrorToMessage } from "@/features/auth/store/mapErrors";
import { Button } from "@/shared/ui/Button";
import { FormField } from "@/shared/ui/FormField";
import { Input } from "@/shared/ui/Input";
import { ru } from "@/shared/lib/i18n/ru";

import styles from "./RegisterForm.module.css";

import type { RegisterValues } from "@/features/auth/components/RegisterForm.schema";

type RegisterFormProps = {
  onSuccess: () => void;
};

export function RegisterForm({ onSuccess }: RegisterFormProps): JSX.Element {
  const form = useForm<RegisterValues>({
    resolver: zodResolver(registerSchema),
    defaultValues: { email: "", username: "", password: "" },
  });

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = form;

  async function onSubmit(values: RegisterValues): Promise<void> {
    const ok = await useAuthStore.getState().register(values);
    if (!ok) {
      const code = useAuthStore.getState().lastErrorCode;
      applyServerError(code);
      return;
    }
    onSuccess();
  }

  // Распределение кодов по полям (см. 07-ui-contract.md / 02-behavior.md).
  function applyServerError(code: string | null): void {
    if (code === "AUTH-001") {
      setError("email", { message: ru.auth.invalidEmail });
      return;
    }
    if (code === "AUTH-004") {
      setError("email", { message: ru.auth.emailTaken });
      return;
    }
    if (code === "AUTH-002") {
      setError("username", { message: ru.auth.usernameFormat });
      return;
    }
    if (code === "AUTH-005") {
      setError("username", { message: ru.auth.usernameTaken });
      return;
    }
    if (code === "AUTH-003") {
      setError("password", { message: ru.auth.passwordMin });
      return;
    }
    setError("root.serverError", {
      message: mapAuthErrorToMessage(code ?? "INTERNAL"),
    });
  }

  const serverErrorMessage = errors.root?.serverError?.message;

  return (
    <form
      className={styles.form}
      aria-label={ru.auth.registerFormLabel}
      onSubmit={handleSubmit(onSubmit)}
      noValidate
    >
      <FormField
        label={ru.auth.email}
        htmlFor="register-email"
        {...(errors.email?.message !== undefined ? { error: errors.email.message } : {})}
      >
        <Input
          id="register-email"
          type="email"
          autoComplete="email"
          invalid={errors.email !== undefined}
          {...register("email")}
        />
      </FormField>

      <FormField
        label={ru.auth.username}
        htmlFor="register-username"
        {...(errors.username?.message !== undefined
          ? { error: errors.username.message }
          : {})}
      >
        <Input
          id="register-username"
          type="text"
          autoComplete="username"
          invalid={errors.username !== undefined}
          {...register("username")}
        />
      </FormField>

      <FormField
        label={ru.auth.password}
        htmlFor="register-password"
        {...(errors.password?.message !== undefined
          ? { error: errors.password.message }
          : {})}
      >
        <Input
          id="register-password"
          type="password"
          autoComplete="new-password"
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
        {ru.auth.registerBtn}
      </Button>

      <div className={styles.footer}>
        <Link to="/login" className={styles.link}>
          {ru.auth.toLogin}
        </Link>
      </div>
    </form>
  );
}
