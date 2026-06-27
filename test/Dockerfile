ARG GO_VERSION=1.26.3

FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/auth-api ./cmd/api

FROM alpine:3.22
WORKDIR /app

RUN addgroup -S app && adduser -S app -G app
COPY --from=build /out/auth-api /app/auth-api

USER app
EXPOSE 8484
ENTRYPOINT ["/app/auth-api"]
