# Stage 1: Build the Go application
FROM --platform=linux/amd64 golang:1.22.2-bullseye AS builder

# Set the working directory inside the container
WORKDIR /app

# NOTE: Install necessary libraries for SQLite
RUN apt-get update && apt-get install -y \
    gcc \
    musl-dev \
    sqlite3 \
    libsqlite3-dev

# Copy Go modules and download dependencies
COPY go.mod go.sum ./
RUN go mod tidy
RUN go mod download

# Copy the source code
COPY . .

ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64

# Build the application
RUN GOARCH=amd64  go build -ldflags="-s -w" -gcflags=all=-l -o /app/goship-web ./cmd/web/main.go
RUN GOARCH=amd64  go build -ldflags="-s -w" -gcflags=all=-l -o /app/goship-worker ./cmd/worker/main.go
RUN GOARCH=amd64  go build -ldflags="-s -w" -gcflags=all=-l -o /app/goship-seed ./cmd/seed/main.go

# Install asynq tools
RUN go install github.com/hibiken/asynq/tools/asynq@latest

################################################
# Stage 2: Create a smaller runtime image
################################################
FROM ubuntu:22.04

# Install necessary packages
RUN apt-get update && apt-get install -y \
    curl

# Copy the compiled binaries from the builder image
COPY --from=builder /app/goship-web /goship-web
COPY --from=builder /app/goship-worker /goship-worker
COPY --from=builder /app/goship-seed /goship-seed

# Copy asynq tool
COPY --from=builder /go/bin/asynq /usr/local/bin/

# Copy the templates
COPY templates/ /app/templates/

# Optional: Bind to a TCP port (document the ports the application listens on)
EXPOSE 8000
EXPOSE 8080

# Define an entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY config/config.yaml .
COPY service-worker.js /service-worker.js
COPY static /static

# Below is only used if you need to use PWABuilder to make a native Android app
# RUN mkdir pwabuilder-android-wrapper
# COPY pwabuilder-android-wrapper/assetlinks.json pwabuilder-android-wrapper/assetlinks.json 

ENTRYPOINT ["/entrypoint.sh"]

# Clean up any unnecessary files
RUN apt-get purge -y gcc musl-dev libsqlite3-dev && apt-get autoremove -y && apt-get clean