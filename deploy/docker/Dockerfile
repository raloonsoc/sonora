# --- Build stage ---
FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/sonora ./cmd/sonora

# --- Final stage ---
FROM alpine:3.20

RUN apk add --no-cache ffmpeg ca-certificates && mkdir -p /cache/covers /cache/transcodes

COPY --from=build /out/sonora /usr/local/bin/sonora

EXPOSE 4533

ENTRYPOINT ["sonora"]
