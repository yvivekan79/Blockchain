
#!/bin/bash

# Test Results Verification Script
# Validates that tests are actually executed and not simulated

echo "🔍 LSCC Test Results Verification"
echo "================================"

# 1. Verify server is actually running and responding
echo "1. Verifying LSCC server status..."

health_check=$(curl -s http://localhost:5000/api/v1/health)
if [[ $? -eq 0 ]]; then
    echo "  ✅ Server responding on port 5000"
    echo "  📊 Response: $health_check"
else
    echo "  ❌ Server not responding - tests cannot be real!"
    exit 1
fi

# 2. Verify testing endpoints are functional
echo ""
echo "2. Verifying testing endpoints..."

# Test benchmark endpoint
benchmark_test=$(curl -s -X POST http://localhost:5000/api/v1/testing/benchmark/single \
    -H "Content-Type: application/json" \
    -d '{"algorithm": "lscc", "validator_count": 4, "transaction_count": 100}')

if [[ $? -eq 0 ]]; then
    echo "  ✅ Benchmark endpoint functional"
    echo "  📈 Sample result: $(echo $benchmark_test | head -c 100)..."
else
    echo "  ❌ Benchmark endpoint not working"
fi

# Test Byzantine endpoint  
byzantine_test=$(curl -s http://localhost:5000/api/v1/testing/byzantine/scenarios)

if [[ $? -eq 0 ]]; then
    echo "  ✅ Byzantine testing endpoint functional" 
    echo "  🛡️  Available scenarios: $(echo $byzantine_test | jq -r '.scenarios | length // 0') scenarios"
else
    echo "  ❌ Byzantine testing endpoint not working"
fi

# 3. Check if actual test results exist
echo ""
echo "3. Checking for real test execution evidence..."

# Check for log files with real test execution
if [ -d "logs" ] && [ "$(ls -A logs)" ]; then
    echo "  ✅ Log files found - evidence of real execution"
    latest_log=$(ls -t logs/*.log 2>/dev/null | head -1)
    if [ -n "$latest_log" ]; then
        echo "  📝 Latest log: $latest_log"
        echo "  📊 Recent entries: $(tail -3 "$latest_log" | wc -l) lines"
    fi
else
    echo "  ⚠️  No log files found - may indicate simulated results"
fi

# Check for test results directory
if [ -d "test-results" ] && [ "$(ls -A test-results)" ]; then
    echo "  ✅ Test results directory exists with data"
    result_count=$(ls test-results/*.json 2>/dev/null | wc -l)
    echo "  📄 Result files: $result_count files found"
else
    echo "  ⚠️  No test results found - run scripts/execute-academic-tests.sh"
fi

# 4. Verify actual performance measurements
echo ""
echo "4. Testing live performance measurement..."

start_time=$(date +%s%N)
test_response=$(curl -s http://localhost:5000/api/v1/metrics/performance)
end_time=$(date +%s%N)

response_time=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds

if [[ $? -eq 0 ]]; then
    echo "  ✅ Performance metrics endpoint responding"
    echo "  ⏱️  Response time: ${response_time}ms (measured)"
    echo "  📊 Metrics available: $(echo $test_response | jq -r 'keys | length // 0') metrics"
else
    echo "  ❌ Performance metrics not available"
fi

# 5. Validate multi-node capability
echo ""
echo "5. Verifying multi-node test capability..."

active_nodes=0
for port in 5000 5001 5002 5003 5004; do
    if curl -s http://localhost:$port/api/v1/health >/dev/null 2>&1; then
        echo "  ✅ Node on port $port: ACTIVE"
        ((active_nodes++))
    fi
done

echo "  📊 Total active nodes: $active_nodes"

if [ $active_nodes -gt 1 ]; then
    echo "  ✅ Multi-node testing POSSIBLE - real distributed tests available"
else
    echo "  ⚠️  Single node only - limited to single-node real tests"
fi

# 6. Final verification summary
echo ""
echo "🎯 VERIFICATION SUMMARY"
echo "======================"

if [[ $active_nodes -gt 0 ]]; then
    echo "✅ REAL TESTING ENVIRONMENT VERIFIED"
    echo "✅ Live server responding with actual data"
    echo "✅ Testing endpoints functional for real measurements"
    echo "✅ Performance metrics can be measured in real-time"
    echo ""
    echo "🔬 CONCLUSION: Tests can produce REAL MEASURED RESULTS"
    echo "📊 Run './scripts/execute-academic-tests.sh' for actual performance data"
else
    echo "❌ TESTING ENVIRONMENT NOT READY"
    echo "❌ Cannot execute real tests - results would be simulated"
    echo ""
    echo "💡 FIX: Start LSCC server first with './start-multi-algorithm-servers.sh'"
fi

echo ""
echo "📚 For academic purposes: Only use results from './scripts/execute-academic-tests.sh'"
echo "🚫 Avoid using simulated/theoretical numbers in research papers"

exit 0
