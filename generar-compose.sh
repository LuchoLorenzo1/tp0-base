#!/bin/bash

if [ $# -ne 2 ]; then
  echo "Usage: $0 <file> <number_of_clients>"
  exit 1
fi

if [ -d $1 ]; then
  echo "First argument must not be a directory"
  exit 1
fi

if ! [[ $2 =~ ^[0-9]+$ ]]; then
  echo "Second argument must be a number"
  exit 1
fi

file="name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=1
      - LOGGING_LEVEL=DEBUG
    networks:
      - testing_net"

echo "$file" > $1

for i in $(seq 1 $2)
do
  client="
  client$i:
  	container_name: client$i
	image: client:latest
	entrypoint: /client
	environment:
		- CLI_ID=$i
		- CLI_LOG_LEVEL=DEBUG
	networks:
		- testing_net
	depends_on:
		- server"

	echo "$client" >> $1
done


networks="networks:
	testing_net:
		ipam:
			driver: default
			config:
				- subnet: 172.25.125.0/24"

echo "$networks" >> $1
