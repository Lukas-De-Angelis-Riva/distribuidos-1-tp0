#!/bin/sh

response=$(echo "PING" | nc server $SERVER_PORT)

if [ "$response" == "PING" ]; then
	echo "Test passed"
else
	echo "Test failed"
fi