FROM golang:1.18-alpine3.15 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go build -ldflags '-s -w' ./cmd/ionscale

FROM alpine:3.15
WORKDIR /app
COPY --from=build /app/ionscale /usr/local/bin/ionscale
VOLUME ["/data/conf","/data/policies","/data/db"]
ENTRYPOINT ["/usr/local/bin/ionscale"]
CMD ["server", "--config", "/data/conf/ionscale.yaml"]