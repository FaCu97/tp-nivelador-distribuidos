#!/bin/bash

NETWORK_NAME="tp-nivelador-distribuidos_default"

MESSAGE="Hello World"

# Wait for all client containers to finish
for client in client_0 client_1 client_2 client_3 client_4; do
    if docker inspect "$client" >/dev/null 2>&1; then
        while [ "$(docker inspect -f '{{.State.Running}}' "$client")" = "true" ]; do
            sleep 1
        done
    fi
done

RESPONSE=$(docker run --rm -i \
  --network "$NETWORK_NAME" \
  alpine sh -c \
  "apk add --no-cache netcat-openbsd >/dev/null && \
   printf '$MESSAGE\n' | nc -N -w 1 server 5678")

if [ "$RESPONSE" == "$MESSAGE" ]; then
    echo "Network test passed: Received expected response from server."
else
    echo "Network test failed: Expected '$MESSAGE' but got '$RESPONSE'."
    exit 1
fi