FROM --platform=${BUILDPLATFORM:-linux/amd64} alpine:3.16.2

COPY ionscale /usr/local/bin/ionscale

RUN mkdir -p /data/ionscale
WORKDIR /data/ionscale

ENTRYPOINT ["/usr/local/bin/ionscale"]