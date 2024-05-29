FROM --platform=${BUILDPLATFORM:-linux/amd64} alpine:3.20.0

COPY ionscale /usr/local/bin/ionscale

RUN mkdir -p /data/ionscale
WORKDIR /data/ionscale

ENTRYPOINT ["/usr/local/bin/ionscale"]