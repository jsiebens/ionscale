FROM alpine:3.14.0

ARG TAILSCALE_VERSION

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

WORKDIR /app

ENV TSFILE=tailscale_${TAILSCALE_VERSION}_amd64.tgz
RUN wget https://pkgs.tailscale.com/stable/${TSFILE} && tar xzf ${TSFILE} --strip-components=1
RUN mkdir -p /var/run/tailscale /var/cache/tailscale /var/lib/tailscale /.cache
