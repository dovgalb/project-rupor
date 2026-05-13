// Единый logger. В dev-режиме пишет debug/info в console.log;
// в проде — только warn/error. Не логирует токены и тела auth-запросов (R-09).
const isDev = import.meta.env.DEV;

export const logger = {
  debug: (...args: unknown[]): void => {
    if (isDev) {
      // eslint-disable-next-line no-console -- единственное допустимое место для debug-логов
      console.log("[debug]", ...args);
    }
  },
  info: (...args: unknown[]): void => {
    if (isDev) {
      // eslint-disable-next-line no-console -- единственное допустимое место для info-логов
      console.log("[info]", ...args);
    }
  },
  warn: (...args: unknown[]): void => {
    console.warn(...args);
  },
  error: (...args: unknown[]): void => {
    console.error(...args);
  },
};
