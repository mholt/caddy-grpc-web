ARG CADDY_VER=2.6.1

FROM caddy:${CADDY_VER}-builder-alpine AS builder

RUN xcaddy build \
  --with github.com/mholt/caddy-grpc-web

FROM caddy:${CADDY_VER}-alpine

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
