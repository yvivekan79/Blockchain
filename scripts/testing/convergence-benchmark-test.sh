#!/bin/bash

# Convergence Benchmark Test
# Tests convergence time and TPS across distributed nodes

set -e

# Configuration
NODES=("192.168.50.147" "192.168.50.148" "192.168.50.149" "192.168.50.150")
PORT=5000
TRANSACTIONS_PER_NODE=150

TEST_ID="convergence_benchmark_$(date +%Y%m%d_%H%M%S)"
LOG_FILE="${TEST_ID}.log"
RESULTS_FILE="${TEST_ID}_results.json"

echo "=== Convergence Benchmark Test ===" | tee $LOG_FILE
echo "Test ID: $TEST_ID" | tee -a $LOG_FILE
echo "Configuration: ${TRANSACTIONS_PER_NODE} transactions per node" | tee -a $LOG_FILE
echo "Nodes: ${#NODES[@]}" | tee -a $LOG_FILE
echo "Start time: $(date)" | tee -a $LOG_FILE
echo "" | tee -a $LOG_FILE

# Function to safely extract JSON values
extract_json_value() {
    local json="$1"
    local key="$2"
    local default="$3"
    
    if command -v jq >/dev/null 2>&1; then
        echo "$json" | jq -r "${key} // \"${default}\"" 2>/dev/null || echo "$default"
    else
        echo "$default"
    fi
}

# Function to check endpoint health
check_endpoint_health() {
    local node=$1
    local response=$(curl -s --connect-timeout 5 --max-time 10 "http://${node}:${PORT}/health" 2>/dev/null || echo "")
    
    if [[ "$response" == *"healthy"* ]] || [[ "$response" == *"ok"* ]] || [[ "$response" == *"status"* ]]; then
        return 0
    else
        return 1
    fi
}

# Function to get initial blockchain state
get_blockchain_state() {
    local node=$1
    local response=$(curl -s --connect-timeout 10 --max-time 15 "http://${node}:${PORT}/api/v1/blockchain/info" 2>/dev/null || echo '{"error": "no_response"}')
    echo "$response"
}

# Function to inject transactions
inject_transactions() {
    local node=$1
    local count=$2
    
    echo "  Injecting $count transactions to $node..." | tee -a $LOG_FILE
    
    local payload="{\"count\": ${count}}"
    
    local response=$(curl -s --connect-timeout 15 --max-time 30 \
        -X POST "http://${node}:${PORT}/api/v1/transaction-injection/inject-batch" \
        -H "Content-Type: application/json" \
        -d "$payload" 2>/dev/null || echo '{"error": "injection_failed"}')
    
    if echo "$response" | jq empty 2>/dev/null; then
        local successful=$(extract_json_value "$response" ".successful // .injected" "0")
        local duration=$(extract_json_value "$response" ".duration_ms" "0")
        local tps=$(extract_json_value "$response" ".actual_tps" "0")
        
        echo "    Result: ${successful} successful, ${duration}ms, ${tps} TPS" | tee -a $LOG_FILE
        echo "$response"
    else
        echo "    Result: Injection failed - invalid response" | tee -a $LOG_FILE
        echo '{"error": "invalid_response", "successful": 0}'
    fi
}

# Phase 1: Health Check
echo "Phase 1: Health Check" | tee -a $LOG_FILE
echo "=====================" | tee -a $LOG_FILE

HEALTHY_NODES=()
for node in "${NODES[@]}"; do
    echo -n "  Node $node: " | tee -a $LOG_FILE
    if check_endpoint_health "$node"; then
        echo "HEALTHY" | tee -a $LOG_FILE
        HEALTHY_NODES+=("$node")
    else
        echo "NOT RESPONDING" | tee -a $LOG_FILE
    fi
done

echo "" | tee -a $LOG_FILE
echo "Healthy nodes: ${#HEALTHY_NODES[@]}/${#NODES[@]}" | tee -a $LOG_FILE

if [ ${#HEALTHY_NODES[@]} -eq 0 ]; then
    echo "ERROR: No healthy nodes available" | tee -a $LOG_FILE
    exit 1
fi

# Phase 2: Initial State
echo "" | tee -a $LOG_FILE
echo "Phase 2: Initial State" | tee -a $LOG_FILE
echo "======================" | tee -a $LOG_FILE

declare -A INITIAL_HEIGHTS
for node in "${HEALTHY_NODES[@]}"; do
    state=$(get_blockchain_state "$node")
    height=$(extract_json_value "$state" ".chain_height // .height" "0")
    INITIAL_HEIGHTS[$node]=$height
    echo "  Node $node: Height = $height" | tee -a $LOG_FILE
done

# Phase 3: Transaction Injection
echo "" | tee -a $LOG_FILE
echo "Phase 3: Transaction Injection" | tee -a $LOG_FILE
echo "==============================" | tee -a $LOG_FILE

START_TIME=$(date +%s%N)
TOTAL_INJECTED=0

for node in "${HEALTHY_NODES[@]}"; do
    result=$(inject_transactions "$node" "$TRANSACTIONS_PER_NODE")
    successful=$(extract_json_value "$result" ".successful // .injected" "0")
    TOTAL_INJECTED=$((TOTAL_INJECTED + successful))
done

INJECTION_END_TIME=$(date +%s%N)
INJECTION_TIME_MS=$(( (INJECTION_END_TIME - START_TIME) / 1000000 ))

echo "" | tee -a $LOG_FILE
echo "Total injected: $TOTAL_INJECTED transactions in ${INJECTION_TIME_MS}ms" | tee -a $LOG_FILE

# Phase 4: Convergence Monitoring
echo "" | tee -a $LOG_FILE
echo "Phase 4: Convergence Monitoring" | tee -a $LOG_FILE
echo "================================" | tee -a $LOG_FILE

CONVERGED=false
ROUNDS=0
MAX_ROUNDS=30

while [ "$CONVERGED" = false ] && [ $ROUNDS -lt $MAX_ROUNDS ]; do
    ROUNDS=$((ROUNDS + 1))
    sleep 2
    
    echo "Round $ROUNDS:" | tee -a $LOG_FILE
    
    ALL_CONVERGED=true
    for node in "${HEALTHY_NODES[@]}"; do
        state=$(get_blockchain_state "$node")
        height=$(extract_json_value "$state" ".chain_height // .height" "0")
        txs=$(extract_json_value "$state" ".total_transactions // .transaction_count" "0")
        
        initial_height=${INITIAL_HEIGHTS[$node]}
        height_diff=$((height - initial_height))
        
        printf "  %-18s Height: %-4s (+%-2s) Txs: %-6s\n" "$node" "$height" "$height_diff" "$txs" | tee -a $LOG_FILE
        
        if [ $height_diff -lt 1 ]; then
            ALL_CONVERGED=false
        fi
    done
    
    if [ "$ALL_CONVERGED" = true ]; then
        CONVERGED=true
    fi
done

END_TIME=$(date +%s%N)
TOTAL_TIME_MS=$(( (END_TIME - START_TIME) / 1000000 ))
TOTAL_TIME_S=$((TOTAL_TIME_MS / 1000))

# Phase 5: Results
echo "" | tee -a $LOG_FILE
echo "=== BENCHMARK RESULTS ===" | tee -a $LOG_FILE
echo "=========================" | tee -a $LOG_FILE
echo "Test ID: $TEST_ID" | tee -a $LOG_FILE
echo "Nodes tested: ${#HEALTHY_NODES[@]}" | tee -a $LOG_FILE
echo "Total transactions: $TOTAL_INJECTED" | tee -a $LOG_FILE
echo "Injection time: ${INJECTION_TIME_MS}ms" | tee -a $LOG_FILE
echo "Total convergence time: ${TOTAL_TIME_S}s" | tee -a $LOG_FILE
echo "Convergence achieved: $CONVERGED" | tee -a $LOG_FILE

if [ $TOTAL_TIME_MS -gt 0 ]; then
    TPS=$(echo "scale=2; $TOTAL_INJECTED * 1000 / $TOTAL_TIME_MS" | bc -l 2>/dev/null || echo "N/A")
    echo "Average TPS: $TPS" | tee -a $LOG_FILE
fi

# Save results to JSON
cat > "$RESULTS_FILE" << EOF
{
  "test_id": "$TEST_ID",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "nodes_tested": ${#HEALTHY_NODES[@]},
  "transactions_per_node": $TRANSACTIONS_PER_NODE,
  "total_transactions": $TOTAL_INJECTED,
  "injection_time_ms": $INJECTION_TIME_MS,
  "total_time_ms": $TOTAL_TIME_MS,
  "convergence_achieved": $CONVERGED,
  "rounds_to_converge": $ROUNDS
}
EOF

echo "" | tee -a $LOG_FILE
echo "Results saved to: $RESULTS_FILE" | tee -a $LOG_FILE
echo "Log saved to: $LOG_FILE" | tee -a $LOG_FILE

exit 0
