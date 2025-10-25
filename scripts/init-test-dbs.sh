#!/bin/bash
set -e

echo "Waiting for Postgres to be ready..."
max_retries=30
retry_count=0

while ! docker exec postgres-test pg_isready -U postgres > /dev/null 2>&1; do
  retry_count=$((retry_count + 1))
  if [ $retry_count -ge $max_retries ]; then
    echo "Error: Postgres did not become ready in time"
    exit 1
  fi
  echo "Waiting for Postgres... ($retry_count/$max_retries)"
  sleep 1
done

echo "Postgres is ready!"

echo "Waiting for ClickHouse to be ready..."
retry_count=0

while ! docker exec clickhouse-test clickhouse-client --query "SELECT 1" > /dev/null 2>&1; do
  retry_count=$((retry_count + 1))
  if [ $retry_count -ge $max_retries ]; then
    echo "Error: ClickHouse did not become ready in time"
    exit 1
  fi
  echo "Waiting for ClickHouse... ($retry_count/$max_retries)"
  sleep 1
done

echo "ClickHouse is ready!"

echo "Waiting for NATS to be ready..."
retry_count=0

while ! docker logs nats-test 2>&1 | grep -q "Server is ready"; do
  retry_count=$((retry_count + 1))
  if [ $retry_count -ge $max_retries ]; then
    echo "Error: NATS did not become ready in time"
    exit 1
  fi
  echo "Waiting for NATS... ($retry_count/$max_retries)"
  sleep 1
done

echo "NATS is ready!"

# Create test databases
echo "Creating zori_test database in Postgres..."
docker exec postgres-test psql -U postgres -c "DROP DATABASE IF EXISTS zori_test;" || true
docker exec postgres-test psql -U postgres -c "CREATE DATABASE zori_test;"

echo "Creating zori_test database in ClickHouse..."
docker exec clickhouse-test clickhouse-client --query "DROP DATABASE IF EXISTS zori_test;" || true
docker exec clickhouse-test clickhouse-client --query "CREATE DATABASE zori_test;"

echo "Test databases created successfully!"
