#!/bin/bash
set -e

# Function to check if PostgreSQL is ready
check_postgres() {
  pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"
}

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to start..."
until check_postgres; do
  echo "PostgreSQL is unavailable - sleeping"
  sleep 2
done

# Create the extension
echo "PostgreSQL is up - creating extension"
psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "CREATE EXTENSION IF NOT EXISTS vector;"
