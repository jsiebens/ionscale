FROM alpine:3.16.2
COPY ionscale /usr/local/bin/ionscale
ENTRYPOINT ["/usr/local/bin/ionscale"]