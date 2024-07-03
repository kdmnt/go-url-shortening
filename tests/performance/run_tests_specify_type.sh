#!/bin/bash

# Use the BASE_URL and TEST_TYPE provided by docker-compose
# If not set, it will use default values

BASE_URL=${BASE_URL:-"http://localhost:3000"}
TEST_TYPE=${TEST_TYPE:-"LOAD"}

# Function to run a specific test
run_test() {
    local test_type=$1
    local test_file=$2

    echo "Running $test_type test for $test_file"
    k6 run -e TEST_TYPE=$test_type -e BASE_URL=$BASE_URL $test_file
    echo "Finished $test_type test for $test_file"
    echo "-----------------------------"
}

# Run all test files with the specified TEST_TYPE
for test_file in /scripts/test_*.js; do
    run_test $TEST_TYPE $test_file
done
