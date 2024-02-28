ARG TAILSCALE_VERSION

FROM tailscale/tailscale:${TAILSCALE_VERSION} as src
FROM alpine:latest

RUN apk update && apk add ca-certificates openssh curl && rm -rf /var/cache/apk/*
COPY ./tests/config/ca.pem /usr/local/share/ca-certificates/
RUN update-ca-certificates

WORKDIR /app
COPY --from=src /usr/local/bin/tailscale .
COPY --from=src /usr/local/bin/tailscaled .

RUN mkdir -p /var/run/tailscale /var/cache/tailscale /var/lib/tailscale /.cache
