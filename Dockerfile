FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/auth-api ./main.go

FROM alpine:3.20

WORKDIR /app
RUN apk add --no-cache ca-certificates && adduser -D -H appuser

COPY --from=builder /out/auth-api /app/auth-api

USER appuser

EXPOSE 8080

ENV PORT=8080
CMD ["/app/auth-api"]
