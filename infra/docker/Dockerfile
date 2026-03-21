# Stage 1: Build the Go application
FROM golang:1.25.6 AS builder

# Set the working directory inside the container
WORKDIR /app

# NOTE: Install necessary libraries for SQLite
RUN apt-get update && apt-get install -y \
    gcc \
    musl-dev \
    sqlite3 \
    libsqlite3-dev

# Copy the source code
COPY . .

ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOWORK=off

# Build the application
WORKDIR /app
RUN go mod download
RUN go build -ldflags="-s -w" -gcflags=all=-l -o /app-web ./cmd/web
RUN go build -ldflags="-s -w" -gcflags=all=-l -o /app-worker ./cmd/worker
RUN go build -ldflags="-s -w" -gcflags=all=-l -o /app-seed ./cmd/seed

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
COPY --from=builder /app-web /goship-web
COPY --from=builder /app-worker /goship-worker
COPY --from=builder /app-seed /goship-seed

# Copy asynq tool
COPY --from=builder /go/bin/asynq /usr/local/bin/

# Optional: Bind to a TCP port (document the ports the application listens on)
EXPOSE 8000
EXPOSE 8080

# Define an entrypoint script
COPY infra/docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY static/ /static/

# Below is only used if you need to use PWABuilder to make a native Android app
# RUN mkdir pwabuilder-android-wrapper
# COPY pwabuilder-android-wrapper/assetlinks.json pwabuilder-android-wrapper/assetlinks.json 

ENTRYPOINT ["/entrypoint.sh"]

# Clean up any unnecessary files
RUN apt-get purge -y gcc musl-dev libsqlite3-dev && apt-get autoremove -y && apt-get clean
