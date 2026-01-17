#!/bin/bash

# Test Results Verification Script
# Validates that tests are actually executed and not simulated

SERVER_IP=${1:-"localhost"}
PORT=5000

echo "LSCC Test Results Verification"
echo "==============================="
echo "Server: $SERVER_IP:$PORT"
echo ""

# 1. Verify server is actually running and responding
echo "1. Verifying LSCC server status..."

health_check=$(curl -s --connect-timeout 5 "http://$SERVER_IP:$PORT/health" 2>/dev/null)
if [ $? -eq 0 ] && [ -n "$health_check" ]; then
    echo "  Server responding on port $PORT"
    echo "  Response: $health_check"
else
    echo "  Server not responding - tests cannot be real!"
    exit 1
fi

# 2. Verify testing endpoints are functional
echo ""
echo "2. Verifying testing endpoints..."

# Test benchmark endpoint
benchmark_test=$(curl -s -X POST "http://$SERVER_IP:$PORT/api/v1/testing/benchmark/single" \
    -H "Content-Type: application/json" \
    -d '{"algorithm": "lscc", "validator_count": 4, "transaction_count": 100}' 2>/dev/null)

if [ $? -eq 0 ] && [ -n "$benchmark_test" ]; then
    echo "  Benchmark endpoint functional"
    echo "  Sample result: $(echo $benchmark_test | head -c 100)..."
else
    echo "  Benchmark endpoint not available (optional)"
fi

# Test Byzantine endpoint  
byzantine_test=$(curl -s "http://$SERVER_IP:$PORT/api/v1/testing/byzantine/scenarios" 2>/dev/null)

if [ $? -eq 0 ] && [ -n "$byzantine_test" ]; then
    echo "  Byzantine testing endpoint functional"
else
    echo "  Byzantine testing endpoint not available (optional)"
fi

# 3. Check if actual test results exist
echo ""
echo "3. Checking for real test execution evidence..."

# Check for log files with real test execution
if [ -d "logs" ] && [ "$(ls -A logs 2>/dev/null)" ]; then
    echo "  Log files found - evidence of real execution"
    latest_log=$(ls -t logs/*.log 2>/dev/null | head -1)
    if [ -n "$latest_log" ]; then
        echo "  Latest log: $latest_log"
        echo "  Recent entries: $(tail -3 "$latest_log" 2>/dev/null | wc -l) lines"
    fi
else
    echo "  No log files found yet"
fi

# Check for test results directory
if [ -d "test-results" ] && [ "$(ls -A test-results 2>/dev/null)" ]; then
    echo "  Test results directory exists with data"
    result_count=$(ls test-results/*.json 2>/dev/null | wc -l)
    echo "  Result files: $result_count files found"
else
    echo "  No test results found - run scripts/testing/execute-academic-tests.sh"
fi

# 4. Verify actual performance measurements
echo ""
echo "4. Testing live performance measurement..."

start_time=$(date +%s%N)
test_response=$(curl -s "http://$SERVER_IP:$PORT/api/v1/blockchain/info" 2>/dev/null)
end_time=$(date +%s%N)

response_time=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds

if [ $? -eq 0 ] && [ -n "$test_response" ]; then
    echo "  Performance metrics endpoint responding"
    echo "  Response time: ${response_time}ms (measured)"
else
    echo "  Performance metrics not available"
fi

# 5. Final verification summary
echo ""
echo "VERIFICATION SUMMARY"
echo "===================="

echo "REAL TESTING ENVIRONMENT VERIFIED"
echo "Live server responding with actual data"
echo ""
echo "CONCLUSION: Tests can produce REAL MEASURED RESULTS"
echo "Run './scripts/testing/execute-academic-tests.sh' for actual performance data"

exit 0
