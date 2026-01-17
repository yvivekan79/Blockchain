#!/bin/bash

# Multi-Algorithm Convergence Test Script
# Tests convergence across 4 nodes running 4 algorithms each

set -e

# Configuration
NODES=(
    "192.168.50.147"
    "192.168.50.148" 
    "192.168.50.149"
    "192.168.50.150"
)

ALGORITHMS=("pow" "pos" "pbft" "lscc")
PORTS=(5001 5002 5003 5004)

echo "=== Multi-Algorithm Convergence Test ==="
echo "Testing convergence across ${#NODES[@]} nodes with ${#ALGORITHMS[@]} algorithms each"
echo "Total endpoints: $(( ${#NODES[@]} * ${#ALGORITHMS[@]} ))"
echo "Start time: $(date)"
echo

# Function to test single endpoint
test_endpoint() {
    local node_ip=$1
    local port=$2
    local algo=$3
    local node_num=$4

    echo "Testing Node ${node_num} ${algo^^} (${node_ip}:${port})..."

    # Test basic connectivity
    if ! curl -s --connect-timeout 5 "http://${node_ip}:${port}/api/v1/blockchain/info" > /dev/null; then
        echo "  ❌ Not responding"
        return 1
    fi

    # Get blockchain info
    local info=$(curl -s "http://${node_ip}:${port}/api/v1/blockchain/info")
    local height=$(echo "$info" | jq -r '.chain_height // 0')
    local transactions=$(echo "$info" | jq -r '.total_transactions // 0')
    local algorithm=$(echo "$info" | jq -r '.consensus_algorithm // "unknown"')

    echo "  ✅ Height: ${height}, Transactions: ${transactions}, Algorithm: ${algorithm}"

    return 0
}

# Function to inject transactions across all endpoints
inject_transactions() {
    local tx_count=$1
    echo "🚀 Injecting ${tx_count} transactions across all endpoints..."

    local total_injected=0
    local successful_endpoints=0

    for i in "${!NODES[@]}"; do
        local node_ip=${NODES[$i]}
        local node_num=$((i + 1))

        for j in "${!ALGORITHMS[@]}"; do
            local algo=${ALGORITHMS[$j]}
            local port=${PORTS[$j]}

            echo "  Injecting to Node ${node_num} ${algo^^}..."

            local result=$(curl -s -X POST "http://${node_ip}:${port}/api/v1/transaction-injection/inject-batch" \
                -H "Content-Type: application/json" \
                -d "{\"count\": ${tx_count}, \"rate_per_second\": 100}" 2>/dev/null || echo '{"error": "failed"}')

            local successful=$(echo "$result" | jq -r '.successful // 0')
            local tps=$(echo "$result" | jq -r '.actual_tps // 0')

            if [ "$successful" -gt 0 ]; then
                echo "    ✅ Injected ${successful} transactions (${tps} TPS)"
                total_injected=$((total_injected + successful))
                successful_endpoints=$((successful_endpoints + 1))
            else
                echo "    ❌ Injection failed"
            fi
        done
    done

    echo "📊 Total injected: ${total_injected} transactions across ${successful_endpoints} endpoints"
}

# Function to check convergence
check_convergence() {
    echo "🔍 Checking blockchain convergence across all nodes..."

    local heights=()
    local total_transactions=()
    local converged=true

    for i in "${!NODES[@]}"; do
        local node_ip=${NODES[$i]}
        local node_num=$((i + 1))

        echo "  Node ${node_num} (${node_ip}):"

        for j in "${!ALGORITHMS[@]}"; do
            local algo=${ALGORITHMS[$j]}
            local port=${PORTS[$j]}

            local info=$(curl -s "http://${node_ip}:${port}/api/v1/blockchain/info" 2>/dev/null || echo '{}')
            local height=$(echo "$info" | jq -r '.chain_height // 0')
            local transactions=$(echo "$info" | jq -r '.total_transactions // 0')
            local peers=$(echo "$info" | jq -r '.network_peers // 0')

            echo "    ${algo^^}: Height ${height}, Transactions ${transactions}, Peers ${peers}"

            heights+=($height)
            total_transactions+=($transactions)
        done
        echo
    done

    # Analyze convergence
    local min_height=$(printf '%s\n' "${heights[@]}" | sort -n | head -n1)
    local max_height=$(printf '%s\n' "${heights[@]}" | sort -n | tail -n1)
    local height_diff=$((max_height - min_height))

    local min_tx=$(printf '%s\n' "${total_transactions[@]}" | sort -n | head -n1)
    local max_tx=$(printf '%s\n' "${total_transactions[@]}" | sort -n | tail -n1)
    local tx_diff=$((max_tx - min_tx))

    echo "📈 Convergence Analysis:"
    echo "  Block Height Range: ${min_height} - ${max_height} (diff: ${height_diff})"
    echo "  Transaction Range: ${min_tx} - ${max_tx} (diff: ${tx_diff})"

    if [ $height_diff -le 2 ] && [ $tx_diff -le 100 ]; then
        echo "  ✅ Good convergence achieved"
        return 0
    else
        echo "  ⚠️  Convergence needs improvement"
        return 1
    fi
}

# Function to run comprehensive convergence test
run_convergence_test() {
    local test_duration=$1
    local batch_size=$2

    echo "🧪 Running ${test_duration}s convergence test with ${batch_size} transaction batches..."

    local start_time=$(date +%s)
    local end_time=$((start_time + test_duration))
    local round=1

    while [ $(date +%s) -lt $end_time ]; do
        echo "--- Round ${round} ---"

        # Inject transactions
        inject_transactions $batch_size

        # Wait for processing
        echo "⏳ Waiting 30 seconds for processing..."
        sleep 30

        # Check convergence
        if check_convergence; then
            echo "✅ Round ${round} convergence successful"
        else
            echo "⚠️  Round ${round} convergence incomplete"
        fi

        round=$((round + 1))
        echo
    done

    echo "🏁 Convergence test completed after ${test_duration} seconds"
}

# Function to test multi-algorithm performance
test_algorithm_performance() {
    echo "🎯 Testing algorithm-specific performance..."

    local results_file="multi-algorithm-performance-$(date +%Y%m%d_%H%M%S).json"
    echo "{" > $results_file
    echo "  \"test_timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"," >> $results_file
    echo "  \"results\": {" >> $results_file

    local first=true

    for i in "${!NODES[@]}"; do
        local node_ip=${NODES[$i]}
        local node_num=$((i + 1))

        for j in "${!ALGORITHMS[@]}"; do
            local algo=${ALGORITHMS[$j]}
            local port=${PORTS[$j]}

            if [ "$first" = false ]; then
                echo "," >> $results_file
            fi
            first=false

            echo "  Testing ${algo^^} performance on Node ${node_num}..."

            # Run algorithm convergence test
            local convergence_result=$(curl -s -X POST "http://${node_ip}:${port}/api/v1/testing/convergence/all-protocols" \
                -H "Content-Type: application/json" \
                -d '{"transaction_count": 100, "concurrent_algorithms": ["'${algo}'"], "duration_seconds": 10}' 2>/dev/null || echo '{}')

            local convergence_time=$(echo "$convergence_result" | jq -r ".data.test_results.${algo}.convergence_time_ms // 0")
            local success_rate=$(echo "$convergence_result" | jq -r ".data.test_results.${algo}.success_rate // 0")

            echo "    Convergence: ${convergence_time}ms, Success: ${success_rate}%"

            cat >> $results_file << EOF
    "node${node_num}_${algo}": {
      "node": ${node_num},
      "algorithm": "${algo}",
      "endpoint": "${node_ip}:${port}",
      "convergence_time_ms": ${convergence_time},
      "success_rate": ${success_rate}
    }
EOF
        done
    done

    echo "" >> $results_file
    echo "  }" >> $results_file
    echo "}" >> $results_file

    echo "📊 Performance results saved to: ${results_file}"
}

# Main test execution
main() {
    echo "🚀 Starting comprehensive multi-algorithm convergence test..."

    # Test all endpoints
    echo "🔗 Testing endpoint connectivity..."
    local active_endpoints=0

    for i in "${!NODES[@]}"; do
        local node_ip=${NODES[$i]}
        local node_num=$((i + 1))

        for j in "${!ALGORITHMS[@]}"; do
            local algo=${ALGORITHMS[$j]}
            local port=${PORTS[$j]}

            if test_endpoint $node_ip $port $algo $node_num; then
                active_endpoints=$((active_endpoints + 1))
            fi
        done
    done

    echo "📊 Active endpoints: ${active_endpoints}/$(( ${#NODES[@]} * ${#ALGORITHMS[@]} ))"

    if [ $active_endpoints -eq 0 ]; then
        echo "❌ No endpoints responding. Check deployment status."
        exit 1
    fi

    # Run convergence test
    run_convergence_test 120 50  # 2 minutes with 50 tx batches

    # Test algorithm performance
    test_algorithm_performance

    # Final convergence check
    echo "🏆 Final convergence verification..."
    if check_convergence; then
        echo "✅ Multi-algorithm cluster convergence successful!"
    else
        echo "⚠️  Convergence needs optimization"
    fi
}

# Script execution
if [ "$1" = "--quick" ]; then
    echo "🔍 Quick connectivity test..."
    for i in "${!NODES[@]}"; do
        local node_ip=${NODES[$i]}
        local node_num=$((i + 1))
        for j in "${!ALGORITHMS[@]}"; do
            local algo=${ALGORITHMS[$j]}
            local port=${PORTS[$j]}
            test_endpoint $node_ip $port $algo $node_num
        done
    done
elif [ "$1" = "--convergence" ]; then
    check_convergence
elif [ "$1" = "--performance" ]; then
    test_algorithm_performance
else
    main
fi

echo
echo "=== Test Complete ==="
echo "Multi-algorithm convergence testing finished"
echo "Use '--quick' for connectivity test only"
echo "Use '--convergence' for convergence check only"
echo "Use '--performance' for performance testing only"