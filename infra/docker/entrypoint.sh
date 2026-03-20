#!/bin/sh

# Default to running the app if no arguments are given
if [ "$#" -eq 0 ]; then
    exec /goship-web
fi

# Otherwise, run the specified executable
case "$1" in
    web)
        exec /goship-web
        ;;
    worker)
        /goship-worker &
        # asynqmon --port=8080 --redis-addr=localhost:6379 &
        wait
        ;;
    seeder)
        exec /goship-seeder
        ;;
    *)
        echo "Unknown command: $1"
        exit 1
        ;;
esac

# docker run --rm --name asynqmon -e REDIS_URL="redis:164.92.66.136:6379" -p 8080:8080 hibiken/asynqmon