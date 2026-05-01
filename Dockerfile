# Stage 1 — build the binary
FROM golang:1.26-alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o subman .

# Stage 2 — minimal runtime image
FROM alpine:3.20

# ca-certificates: needed for HTTPS calls to Telegram/Twilio
# tzdata: needed so the cron worker fires at the correct local time
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/subman    ./subman
COPY --from=builder /app/templates ./templates

EXPOSE 8080

CMD ["./subman"]
