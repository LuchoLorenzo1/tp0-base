#!/bin/bash

SERVER_CONTAINER_NAME="server"
NETWORK_NAME="tp0_testing_net"

SERVER_PORT=12345
MESSAGE="message"

RESPONSE=$(docker run --rm --network $NETWORK_NAME busybox sh -c "echo $MESSAGE | nc $SERVER_CONTAINER_NAME $SERVER_PORT")

if [ "$RESPONSE" == "$MESSAGE" ]; then
    echo "action: test_echo_server | result: success"
else
    echo "action: test_echo_server | result: fail"
fi
