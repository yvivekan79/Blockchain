#!/bin/bash

# Distributed Convergence Test
# Tests convergence across nodes running the same protocol

set -e

# Configuration
NODES=("192.168.50.147" "192.168.50.148" "192.168.50.149" "192.168.50.150")
PORT=5000
TRANSACTION_COUNT=1000
TEST_ID="distributed_convergence_$(date +%Y%m%d_%H%M%S)"
LOG_FILE="${TEST_ID}.log"

echo "=== Distributed Convergence Test ===" | tee $LOG_FILE
echo "Test ID: $TEST_ID" | tee -a $LOG_FILE
echo "Testing convergence for $TRANSACTION_COUNT transactions" | tee -a $LOG_FILE
echo "Nodes: ${NODES[*]}" | tee -a $LOG_FILE
echo "Start time: $(date)" | tee -a $LOG_FILE
echo "" | tee -a $LOG_FILE

# Phase 1: Check all nodes are healthy
echo "Phase 1: Health Check for All Nodes" | tee -a $LOG_FILE
declare -A NODE_STATUS=()
HEALTHY_NODES=0

for node in "${NODES[@]}"; do
    echo "Checking node at $node:$PORT..." | tee -a $LOG_FILE
    
    if curl -s --connect-timeout 5 "http://$node:$PORT/health" >/dev/null 2>&1; then
        echo "  Node $node is healthy" | tee -a $LOG_FILE
        NODE_STATUS[$node]="healthy"
        HEALTHY_NODES=$((HEALTHY_NODES + 1))
        
        # Get initial blockchain state
        INITIAL_STATE=$(curl -s "http://$node:$PORT/api/v1/blockchain/info" 2>/dev/null || echo '{"error": "no response"}')
        echo "  Initial state: $(echo $INITIAL_STATE | jq -c '.' 2>/dev/null)" | tee -a $LOG_FILE
    else
        echo "  Node $node is not responding" | tee -a $LOG_FILE
        NODE_STATUS[$node]="down"
    fi
done

echo "" | tee -a $LOG_FILE
echo "Healthy nodes: $HEALTHY_NODES/${#NODES[@]}" | tee -a $LOG_FILE

if [ $HEALTHY_NODES -lt 1 ]; then
    echo "ERROR: No healthy nodes for testing" | tee -a $LOG_FILE
    exit 1
fi

# Phase 2: Transaction injection to first healthy node
echo "" | tee -a $LOG_FILE
echo "Phase 2: Transaction Injection" | tee -a $LOG_FILE
START_TIME=$(date +%s)

# Find first healthy node
INJECTION_NODE=""
for node in "${NODES[@]}"; do
    if [ "${NODE_STATUS[$node]}" = "healthy" ]; then
        INJECTION_NODE=$node
        break
    fi
done

echo "Injecting $TRANSACTION_COUNT transactions to $INJECTION_NODE..." | tee -a $LOG_FILE

INJECTION_RESPONSE=$(curl -s -X POST "http://$INJECTION_NODE:$PORT/api/v1/transaction-injection/inject-batch" \
    -H "Content-Type: application/json" \
    -d "{\"count\": $TRANSACTION_COUNT}" 2>/dev/null || echo '{"error": "injection failed"}')

echo "Injection result: $(echo $INJECTION_RESPONSE | jq -c '.' 2>/dev/null)" | tee -a $LOG_FILE

# Phase 3: Monitor convergence across all nodes
echo "" | tee -a $LOG_FILE
echo "Phase 3: Convergence Monitoring" | tee -a $LOG_FILE

CONVERGED=false
ROUNDS=0
MAX_ROUNDS=60  # 2 minutes max wait

while [ "$CONVERGED" = false ] && [ $ROUNDS -lt $MAX_ROUNDS ]; do
    ROUNDS=$((ROUNDS + 1))
    CURRENT_TIME=$(date +%s)
    ELAPSED=$((CURRENT_TIME - START_TIME))
    
    echo "Round $ROUNDS (${ELAPSED}s):" | tee -a $LOG_FILE
    
    TOTAL_CONFIRMED=0
    TOTAL_PENDING=0
    CONVERGENT_NODES=0
    
    for node in "${NODES[@]}"; do
        if [ "${NODE_STATUS[$node]}" = "healthy" ]; then
            BLOCKCHAIN_INFO=$(curl -s "http://$node:$PORT/api/v1/blockchain/info" 2>/dev/null || echo '{"chain_height":0}')
            TX_STATS=$(curl -s "http://$node:$PORT/api/v1/transactions/stats" 2>/dev/null || echo '{"stats":{"confirmed_count":0}}')
            
            CHAIN_HEIGHT=$(echo "$BLOCKCHAIN_INFO" | jq -r '.chain_height // .height // 0' 2>/dev/null)
            CONFIRMED=$(echo "$TX_STATS" | jq -r '.stats.confirmed_count // 0' 2>/dev/null)
            PENDING=$(echo "$TX_STATS" | jq -r '.stats.pending_count // 0' 2>/dev/null)
            
            TOTAL_CONFIRMED=$((TOTAL_CONFIRMED + CONFIRMED))
            TOTAL_PENDING=$((TOTAL_PENDING + PENDING))
            
            if [ "$PENDING" -lt 10 ]; then
                CONVERGENT_NODES=$((CONVERGENT_NODES + 1))
            fi
            
            echo "  $node: Height=$CHAIN_HEIGHT, Confirmed=$CONFIRMED, Pending=$PENDING" | tee -a $LOG_FILE
        fi
    done
    
    # Check if all healthy nodes have converged
    if [ $CONVERGENT_NODES -eq $HEALTHY_NODES ] && [ $TOTAL_CONFIRMED -gt 0 ]; then
        CONVERGED=true
        echo "" | tee -a $LOG_FILE
        echo "CONVERGENCE ACHIEVED in ${ELAPSED}s" | tee -a $LOG_FILE
    fi
    
    sleep 2
done

END_TIME=$(date +%s)
TOTAL_TIME=$((END_TIME - START_TIME))

# Phase 4: Results Summary
echo "" | tee -a $LOG_FILE
echo "=== TEST RESULTS ===" | tee -a $LOG_FILE
echo "Test ID: $TEST_ID" | tee -a $LOG_FILE
echo "Duration: ${TOTAL_TIME}s" | tee -a $LOG_FILE
echo "Transactions: $TRANSACTION_COUNT" | tee -a $LOG_FILE
echo "Healthy Nodes: $HEALTHY_NODES/${#NODES[@]}" | tee -a $LOG_FILE
echo "Convergence: $CONVERGED" | tee -a $LOG_FILE
echo "Log file: $LOG_FILE" | tee -a $LOG_FILE

if [ "$CONVERGED" = true ]; then
    echo "" | tee -a $LOG_FILE
    echo "TEST PASSED - Distributed convergence successful" | tee -a $LOG_FILE
    exit 0
else
    echo "" | tee -a $LOG_FILE
    echo "TEST INCOMPLETE - Convergence not achieved within timeout" | tee -a $LOG_FILE
    exit 1
fi
