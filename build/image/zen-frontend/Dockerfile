# Stage 1: build the SPA bundle.
FROM node:20-alpine AS builder

WORKDIR /build
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY frontend/ ./
# vite.config.ts reads the app version from the repo's project.yaml (one dir up
# from the SPA), so make it reachable — otherwise the bundle reports 'dev'.
COPY project.yaml /project.yaml
RUN npm run build

# Stage 2: serve the bundle through nginx.
FROM nginx:alpine

# Replace the default nginx site config with ours.
RUN rm -f /etc/nginx/conf.d/default.conf
COPY build/image/zen-frontend/nginx.conf /etc/nginx/conf.d/default.conf

COPY --from=builder /build/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
