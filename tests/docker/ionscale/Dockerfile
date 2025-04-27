FROM golang:1.24-bullseye as builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . ./

RUN go build -v -o ionscale cmd/ionscale/main.go

FROM debian:buster-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/ionscale /usr/local/bin/ionscale

ENTRYPOINT ["/usr/local/bin/ionscale"]