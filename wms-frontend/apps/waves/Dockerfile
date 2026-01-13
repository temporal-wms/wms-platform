# syntax=docker/dockerfile:1
FROM node:20-alpine AS builder

WORKDIR /app

ARG APP_NAME=waves

COPY package.json package-lock.json tsconfig.base.json ./
RUN npm ci

COPY packages ./packages
COPY apps ./apps

# Build directly without NX to avoid project graph issues in Docker
WORKDIR /app/apps/${APP_NAME}
RUN npm run build

FROM nginx:alpine

ARG APP_NAME=waves

COPY deploy/nginx/nginx.conf /etc/nginx/nginx.conf
COPY deploy/nginx/default.conf /etc/nginx/conf.d/default.conf

COPY --from=builder /app/apps/${APP_NAME}/dist /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/health || exit 1

CMD ["nginx", "-g", "daemon off;"]
