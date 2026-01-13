# syntax=docker/dockerfile:1
# Multi-stage build for WMS Frontend Microfrontends
# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Build argument to specify which app to build
ARG APP_NAME=shell

# Copy dependency files for better layer caching
COPY package.json package-lock.json tsconfig.base.json ./

# Install dependencies
RUN npm ci

# Copy source files
COPY packages ./packages
COPY apps ./apps

# Build directly without NX to avoid project graph issues in Docker
WORKDIR /app/apps/${APP_NAME}
RUN npm run build

# Production stage
FROM nginx:alpine AS production

# Build argument to specify which app
ARG APP_NAME=shell

# Copy nginx configuration
COPY deploy/nginx/nginx.conf /etc/nginx/nginx.conf
COPY deploy/nginx/default.conf /etc/nginx/conf.d/default.conf

# Copy built assets
COPY --from=builder /app/apps/${APP_NAME}/dist /usr/share/nginx/html

# Expose port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/health || exit 1

CMD ["nginx", "-g", "daemon off;"]
