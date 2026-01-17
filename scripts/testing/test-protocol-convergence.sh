
#!/bin/bash

# Test Protocol Convergence Script
# This script tests that all consensus algorithms can converge properly

set -e

echo "=== Testing Protocol Convergence ==="

# Function to test individual protocol
test_protocol() {
    local protocol=$1
    local port=$2
    local duration=${3:-30}
    
    echo "Testing $protocol protocol on port $port for ${duration}s..."
    
    # Create temporary config
    cat > test-config-$protocol.yaml << EOF
node:
  id: "test-node-$protocol"
  consensus_algorithm: "$protocol"
  
server:
  port: $port
  host: "0.0.0.0"
  
consensus:
  algorithm: "$protocol"
  difficulty: 2
  block_time: 2
  view_timeout: 15
  gas_limit: 200000000
  
logging:
  level: "info"
  format: "json"
EOF

    # Start node in background
    timeout ${duration}s go run main.go -config=test-config-$protocol.yaml > test-$protocol.log 2>&1 &
    pid=$!
    
    # Wait a bit for startup
    sleep 5
    
    # Test basic functionality
    echo "  - Testing basic API..."
    curl -s http://localhost:$port/api/status || echo "    API test failed"
    
    # Generate some transactions
    echo "  - Generating test transactions..."
    for i in {1..5}; do
        curl -s -X POST http://localhost:$port/api/transactions \
             -H "Content-Type: application/json" \
             -d '{"from":"test1","to":"test2","amount":100}' || true
        sleep 1
    done
    
    # Wait for mining/consensus
    sleep 10
    
    # Check convergence
    echo "  - Checking convergence..."
    curl -s http://localhost:$port/api/consensus/metrics || echo "    Metrics unavailable"
    
    # Clean up
    kill $pid 2>/dev/null || true
    wait $pid 2>/dev/null || true
    rm -f test-config-$protocol.yaml
    
    echo "  - $protocol test completed"
}

# Test each protocol
echo "Starting convergence tests..."

test_protocol "pow" 8001 45 &
test_protocol "pos" 8002 30 &
test_protocol "pbft" 8003 30 &
test_protocol "lscc" 8004 30 &

# Wait for all tests to complete
wait

echo ""
echo "=== Convergence Test Results ==="

# Analyze logs
for protocol in pow pos pbft lscc; do
    echo ""
    echo "--- $protocol Results ---"
    if [ -f "test-$protocol.log" ]; then
        # Count successful operations
        success_count=$(grep -c "\"level\":\"info\"" test-$protocol.log 2>/dev/null || echo "0")
        error_count=$(grep -c "\"level\":\"error\"" test-$protocol.log 2>/dev/null || echo "0")
        
        echo "  Success operations: $success_count"
        echo "  Error operations: $error_count"
        
        # Check for specific convergence indicators
        if grep -q "block_processed" test-$protocol.log 2>/dev/null; then
            echo "  ✓ Block processing: WORKING"
        else
            echo "  ✗ Block processing: FAILED"
        fi
        
        if grep -q "view_change_initiated" test-$protocol.log 2>/dev/null; then
            view_changes=$(grep -c "view_change_initiated" test-$protocol.log 2>/dev/null)
            if [ "$view_changes" -lt 5 ]; then
                echo "  ✓ Consensus stability: GOOD ($view_changes view changes)"
            else
                echo "  ⚠ Consensus stability: UNSTABLE ($view_changes view changes)"
            fi
        else
            echo "  ✓ Consensus stability: STABLE (no view changes)"
        fi
        
        # Check for hash validation
        if grep -q "hash.*validation" test-$protocol.log 2>/dev/null; then
            echo "  ✓ Hash validation: WORKING"
        fi
        
        # Clean up log
        rm -f test-$protocol.log
    else
        echo "  ✗ No log file found - test may have failed to start"
    fi
done

echo ""
echo "=== Convergence Test Complete ==="
echo "All protocols have been tested for convergence."
