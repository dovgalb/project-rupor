# === Stage 1: builder ===
# Используем pinned LTS чтобы prod-сборка была воспроизводимой.
FROM node:20-alpine AS builder

WORKDIR /app

# Кешируем npm-зависимости отдельно от исходников: при изменении кода
# слой `npm ci` остаётся валидным.
COPY package.json package-lock.json* ./
RUN npm ci

# Копируем исходники и собираем prod-bundle.
COPY . .
RUN npm run build

# === Stage 2: runtime ===
FROM nginx:1.27-alpine AS runtime

# Собранная SPA → дефолтная статика nginx.
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

# daemon off — стандартная команда nginx в контейнере (логи в stdout/stderr).
CMD ["nginx", "-g", "daemon off;"]
