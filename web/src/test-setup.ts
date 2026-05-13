import "@testing-library/jest-dom/vitest";

// Node 25 экспортирует встроенный globalThis.localStorage (доступный по флагу
// --localstorage-file). Без валидного пути все его методы возвращают undefined,
// и он маскирует jsdom-овский localStorage. Заменяем оба (global и window) на
// in-memory shim — это поведение, которого ожидают тесты shared/api.
class MemoryStorage implements Storage {
  private readonly map = new Map<string, string>();
  get length(): number {
    return this.map.size;
  }
  key(index: number): string | null {
    return Array.from(this.map.keys())[index] ?? null;
  }
  getItem(key: string): string | null {
    return this.map.has(key) ? (this.map.get(key) as string) : null;
  }
  setItem(key: string, value: string): void {
    this.map.set(key, String(value));
  }
  removeItem(key: string): void {
    this.map.delete(key);
  }
  clear(): void {
    this.map.clear();
  }
}

const localShim = new MemoryStorage();
const sessionShim = new MemoryStorage();

Object.defineProperty(globalThis, "localStorage", {
  value: localShim,
  configurable: true,
  writable: true,
});
Object.defineProperty(globalThis, "sessionStorage", {
  value: sessionShim,
  configurable: true,
  writable: true,
});
if (typeof window !== "undefined") {
  Object.defineProperty(window, "localStorage", {
    value: localShim,
    configurable: true,
    writable: true,
  });
  Object.defineProperty(window, "sessionStorage", {
    value: sessionShim,
    configurable: true,
    writable: true,
  });
}
