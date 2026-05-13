import { Toast } from "./Toast";
import { useToastsStore } from "./useToast";
import styles from "./Toaster.module.css";

// Глобальный контейнер тостов. Один раз монтируется в app/App.tsx.
// Подписан на стор; на новый item — рендерит <Toast />.
export function Toaster(): JSX.Element {
  const items = useToastsStore((s) => s.items);

  return (
    <div className={styles.container} aria-label="Уведомления">
      {items.map((item) => (
        <Toast key={item.id} item={item} />
      ))}
    </div>
  );
}
