# Arayüz derlenir ve Caddy ile içeride (HTTP :80) sunulur. HTTPS ve /api
# yönlendirmesi önündeki Traefik'tedir.
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM caddy:2-alpine
COPY Caddyfile /etc/caddy/Caddyfile
COPY --from=build /app/dist /srv
