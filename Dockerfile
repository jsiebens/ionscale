FROM alpine:3.15.4
COPY ionscale /usr/local/bin/ionscale
ENTRYPOINT ["ionscale"]