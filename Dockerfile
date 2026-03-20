FROM golang:alpine AS builder

WORKDIR /app

# Copy entire monorepo (important for go.work)
COPY . .

# Accept build argument for service name
ARG SERVICE
ENV SERVICE=${SERVICE}

# Build the service using BuildKit Cache for massive speedups
WORKDIR /app/${SERVICE}
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
ARG SERVICE

# Copy the built binary
COPY --from=builder /app/${SERVICE}/main .

CMD ["./main"]