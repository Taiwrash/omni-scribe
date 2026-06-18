# ---- Build stage ----
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o omni-scribe .

# ---- Runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /app/omni-scribe /omni-scribe

# Data directory for generated docs (mount a volume here)
ENV DATA_DIR=/data
ENV PORT=8080

EXPOSE 8080

ENTRYPOINT ["/omni-scribe"]
CMD ["serve"]
