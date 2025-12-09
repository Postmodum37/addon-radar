#!/bin/bash
set -e

# Start PostgreSQL container
docker run -d --name addon-radar-db \
  -e POSTGRES_PASSWORD=dev \
  -e POSTGRES_DB=addon_radar \
  -p 5432:5432 \
  postgres:16 || docker start addon-radar-db

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
sleep 3

# Apply schema
echo "Applying schema..."
docker exec -i addon-radar-db psql -U postgres -d addon_radar < sql/schema.sql

echo "Database ready!"
