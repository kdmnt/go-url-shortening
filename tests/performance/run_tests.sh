#!/bin/sh

# Use the BASE_URL provided by docker-compose
# If not set, it will use the default value from docker-compose.yml

# Run load test
echo "Running load test"
k6 run -e TEST_TYPE=LOAD -e BASE_URL=$BASE_URL /scripts/test_create_short_url.js
echo "Finished load test"
echo "-----------------------------"

# Run stress test
echo "Running stress test"
k6 run -e TEST_TYPE=STRESS -e BASE_URL=$BASE_URL /scripts/test_create_short_url.js
echo "Finished stress test"
echo "-----------------------------"

# Run spike test
echo "Running spike test"
k6 run -e TEST_TYPE=SPIKE -e BASE_URL=$BASE_URL /scripts/test_create_short_url.js
echo "Finished spike test"
echo "-----------------------------"
