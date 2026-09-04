#!/bin/bash

client_count=$1

if [ -z "$client_count" ]; then
  echo "Usage: $0 <number_of_clients>"
  exit 1
fi

file_name="docker-compose.yaml"

cat << EOF > "$file_name"
services:
  server:
    build:
      context: ./services/server
      dockerfile: Dockerfile
    container_name: server
    environment:
      - PYTHONUNBUFFERED=1
      - SERVER_HOST=server
      - SERVER_PORT=5678
EOF

for ((i=0; i<client_count; i++));
do
  echo "
  client_$i:
    build:
      context: ./services/client
      dockerfile: Dockerfile
    container_name: client_$i
    depends_on:
      - server
    environment:
      - AGENCY_ID=$i
      - SERVER_HOST=server
      - SERVER_PORT=5678" >> "$file_name"
done