FROM caddy:builder AS builder
RUN xcaddy build --with github.com/circa10a/caddy-geofence

FROM caddy
COPY --from=builder /usr/bin/caddy /usr/bin/caddy
