# Arayüz derlenir, Caddy ile sunulur. Caddy /api isteklerini API'ye aktarır ve
# HTTPS sertifikasını Cloudflare DNS doğrulamasıyla alır.
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM caddy:2-builder AS caddy
RUN xcaddy build --with github.com/caddy-dns/cloudflare

FROM caddy:2-alpine
COPY --from=caddy /usr/bin/caddy /usr/bin/caddy
COPY Caddyfile /etc/caddy/Caddyfile
COPY --from=build /app/dist /srv
