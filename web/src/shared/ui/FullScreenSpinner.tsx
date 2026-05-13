import { Spinner } from "./Spinner";

// Полноэкранный wrapper для guard-loading в Phase 5.
// Заполняет вьюпорт, центрирует Spinner size=lg.
export function FullScreenSpinner(): JSX.Element {
  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        backgroundColor: "var(--color-bg-primary)",
      }}
    >
      <Spinner size="lg" />
    </div>
  );
}
