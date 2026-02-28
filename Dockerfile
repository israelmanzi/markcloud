FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=1 go build -tags "fts5" -o /markcloud-server ./cmd/server/

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

COPY --from=builder /markcloud-server /usr/local/bin/markcloud-server
COPY templates/ /app/templates/
COPY static/ /app/static/

WORKDIR /app

EXPOSE 8080

CMD ["markcloud-server"]
