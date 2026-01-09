# Multi-stage build for WMS Frontend Microfrontends
# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy all files (needed for nx workspace detection)
COPY . .

# Install dependencies
RUN npm ci

# Build argument to specify which app to build
ARG APP_NAME=shell

# Build the specific app using npx nx directly with the @wms scope
RUN npx nx run @wms/${APP_NAME}:build

# Production stage
FROM nginx:alpine AS production

# Copy nginx configuration
COPY deploy/nginx/nginx.conf /etc/nginx/nginx.conf
COPY deploy/nginx/default.conf /etc/nginx/conf.d/default.conf

# Copy built assets
ARG APP_NAME=shell
COPY --from=builder /app/apps/${APP_NAME}/dist /usr/share/nginx/html

# Expose port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/health || exit 1

CMD ["nginx", "-g", "daemon off;"]
