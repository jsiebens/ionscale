ARG TAILSCALE_VERSION

FROM tailscale/tailscale:v${TAILSCALE_VERSION} as src
FROM alpine:3.14.0

RUN apk update && apk add ca-certificates openssh curl && rm -rf /var/cache/apk/*

WORKDIR /app
COPY --from=src /usr/local/bin/tailscale .
COPY --from=src /usr/local/bin/tailscaled .

RUN mkdir -p /var/run/tailscale /var/cache/tailscale /var/lib/tailscale /.cache
