#!/bin/bash

# Distributed Multi-Protocol Convergence Test
# Tests convergence across 4 different consensus algorithms running on separate nodes

set -e

# Protocol endpoints for distributed setup (each node runs one protocol)
declare -A PROTOCOL_ENDPOINTS=(
    ["pow"]="http://192.168.50.147:5001"    # Node 1: PoW
    ["pos"]="http://192.168.50.148:5002"    # Node 2: PoS  
    ["pbft"]="http://192.168.50.149:5003"   # Node 3: PBFT
    ["lscc"]="http://192.168.50.150:5004"   # Node 4: LSCC
)

TRANSACTION_COUNT=1000
TEST_ID="distributed_convergence_$(date +%Y%m%d_%H%M%S)"
LOG_FILE="${TEST_ID}.log"

echo "=== Distributed Multi-Protocol Convergence Test ===" | tee $LOG_FILE
echo "Test ID: $TEST_ID" | tee -a $LOG_FILE
echo "Testing convergence for $TRANSACTION_COUNT transactions across 4 protocols" | tee -a $LOG_FILE
echo "Protocols: PoW(147:5001), PoS(148:5002), PBFT(149:5003), LSCC(150:5004)" | tee -a $LOG_FILE
echo "Start time: $(date)" | tee -a $LOG_FILE
echo "" | tee -a $LOG_FILE

# Phase 1: Check all nodes are healthy
echo "Phase 1: Health Check for All Nodes" | tee -a $LOG_FILE
declare -A NODE_STATUS=()
HEALTHY_NODES=0

for protocol in pow pos pbft lscc; do
    endpoint=${PROTOCOL_ENDPOINTS[$protocol]}
    echo "Checking $protocol node at $endpoint..." | tee -a $LOG_FILE
    
    if curl -s --connect-timeout 5 "$endpoint/health" >/dev/null 2>&1; then
        echo "✅ $protocol node is healthy" | tee -a $LOG_FILE
        NODE_STATUS[$protocol]="healthy"
        HEALTHY_NODES=$((HEALTHY_NODES + 1))
        
        # Get initial blockchain state
        INITIAL_STATE=$(curl -s "$endpoint/api/v1/blockchain/info" 2>/dev/null || echo '{"error": "no response"}')
        echo "[$protocol] Initial state: $INITIAL_STATE" | tee -a $LOG_FILE
    else
        echo "❌ $protocol node is not responding" | tee -a $LOG_FILE
        NODE_STATUS[$protocol]="down"
    fi
done

echo "" | tee -a $LOG_FILE
echo "Healthy nodes: $HEALTHY_NODES/4" | tee -a $LOG_FILE

if [ $HEALTHY_NODES -lt 2 ]; then
    echo "❌ ERROR: Insufficient healthy nodes for distributed testing (need at least 2)" | tee -a $LOG_FILE
    exit 1
fi

# Phase 2: Transaction injection across all healthy nodes
echo "" | tee -a $LOG_FILE
echo "Phase 2: Distributed Transaction Injection" | tee -a $LOG_FILE
START_TIME=$(date +%s)
declare -A INJECTION_RESULTS=()

# Calculate transactions per node
TRANSACTIONS_PER_NODE=$((TRANSACTION_COUNT / HEALTHY_NODES))
echo "Injecting $TRANSACTIONS_PER_NODE transactions per healthy node" | tee -a $LOG_FILE

for protocol in pow pos pbft lscc; do
    if [ "${NODE_STATUS[$protocol]}" = "healthy" ]; then
        endpoint=${PROTOCOL_ENDPOINTS[$protocol]}
        echo "Injecting transactions to $protocol node..." | tee -a $LOG_FILE
        
        INJECTION_RESPONSE=$(curl -s -X POST "$endpoint/api/v1/transaction-injection/inject-batch" \
          -H "Content-Type: application/json" \
          -d "{\"count\": $TRANSACTIONS_PER_NODE, \"rate_per_second\": 50}" 2>/dev/null || echo '{"error": "injection failed"}')
        
        echo "[$protocol] Injection result: $INJECTION_RESPONSE" | tee -a $LOG_FILE
        INJECTION_RESULTS[$protocol]="$INJECTION_RESPONSE"
    fi
done

# Phase 3: Monitor convergence across all protocols
echo "" | tee -a $LOG_FILE
echo "Phase 3: Multi-Protocol Convergence Monitoring" | tee -a $LOG_FILE

CONVERGED=false
ROUNDS=0
MAX_ROUNDS=60  # 2 minutes max wait
MIN_SUCCESS_RATE=90  # At least 90% convergence across all nodes

while [ "$CONVERGED" = false ] && [ $ROUNDS -lt $MAX_ROUNDS ]; do
    ROUNDS=$((ROUNDS + 1))
    CURRENT_TIME=$(date +%s)
    ELAPSED=$((CURRENT_TIME - START_TIME))
    
    echo "Round $ROUNDS (${ELAPSED}s):" | tee -a $LOG_FILE
    
    declare -A PROTOCOL_STATES=()
    TOTAL_CONFIRMED=0
    TOTAL_PENDING=0
    CONVERGENT_PROTOCOLS=0
    
    # Check each healthy protocol
    for protocol in pow pos pbft lscc; do
        if [ "${NODE_STATUS[$protocol]}" = "healthy" ]; then
            endpoint=${PROTOCOL_ENDPOINTS[$protocol]}
            
            # Get current state
            BLOCKCHAIN_INFO=$(curl -s "$endpoint/api/v1/blockchain/info" 2>/dev/null || echo '{"chain_height":0,"total_transactions":0}')
            TX_STATS=$(curl -s "$endpoint/api/v1/transactions/stats" 2>/dev/null || echo '{"stats":{"confirmed_count":0,"pending_count":0}}')
            
            CHAIN_HEIGHT=$(echo "$BLOCKCHAIN_INFO" | jq -r '.chain_height // 0')
            CONFIRMED=$(echo "$TX_STATS" | jq -r '.stats.confirmed_count // 0')
            PENDING=$(echo "$TX_STATS" | jq -r '.stats.pending_count // 0')
            
            TOTAL_CONFIRMED=$((TOTAL_CONFIRMED + CONFIRMED))
            TOTAL_PENDING=$((TOTAL_PENDING + PENDING))
            
            # Check if this protocol has converged (>90% confirmed)
            if [ "$CONFIRMED" -gt 0 ] && [ "$PENDING" -lt $((TRANSACTIONS_PER_NODE / 10)) ]; then
                CONVERGENT_PROTOCOLS=$((CONVERGENT_PROTOCOLS + 1))
            fi
            
            echo "  [$protocol] Height=$CHAIN_HEIGHT, Confirmed=$CONFIRMED, Pending=$PENDING" | tee -a $LOG_FILE
            PROTOCOL_STATES[$protocol]="H:$CHAIN_HEIGHT,C:$CONFIRMED,P:$PENDING"
        fi
    done
    
    # Calculate overall convergence rate
    OVERALL_SUCCESS_RATE=0
    if [ $TOTAL_CONFIRMED -gt 0 ]; then
        OVERALL_SUCCESS_RATE=$(echo "scale=2; $TOTAL_CONFIRMED * 100 / ($TOTAL_CONFIRMED + $TOTAL_PENDING)" | bc -l)
    fi
    
    echo "  Overall: $TOTAL_CONFIRMED confirmed, $TOTAL_PENDING pending (${OVERALL_SUCCESS_RATE}% success)" | tee -a $LOG_FILE
    echo "  Convergent protocols: $CONVERGENT_PROTOCOLS/$HEALTHY_NODES" | tee -a $LOG_FILE
    
    # Check convergence criteria
    if [ "$CONVERGENT_PROTOCOLS" -ge $((HEALTHY_NODES * 3 / 4)) ] && \
       [ "$(echo "$OVERALL_SUCCESS_RATE >= $MIN_SUCCESS_RATE" | bc -l)" -eq 1 ]; then
        CONVERGED=true
        echo "✅ Multi-protocol convergence achieved!" | tee -a $LOG_FILE
    fi
    
    echo "" | tee -a $LOG_FILE
    sleep 2
done

# Phase 4: Final results
END_TIME=$(date +%s)
TOTAL_TIME=$((END_TIME - START_TIME))

echo "=== DISTRIBUTED CONVERGENCE TEST RESULTS ===" | tee -a $LOG_FILE
echo "Test ID: $TEST_ID" | tee -a $LOG_FILE
echo "Total duration: ${TOTAL_TIME} seconds" | tee -a $LOG_FILE
echo "Convergence status: $CONVERGED" | tee -a $LOG_FILE
echo "Healthy nodes: $HEALTHY_NODES/4" | tee -a $LOG_FILE
echo "Convergent protocols: $CONVERGENT_PROTOCOLS/$HEALTHY_NODES" | tee -a $LOG_FILE

if [ "$CONVERGED" = true ]; then
    echo "✅ SUCCESS: Distributed protocols converged successfully" | tee -a $LOG_FILE
    echo "Average processing time: $(echo "scale=2; $TOTAL_TIME / 4" | bc -l) seconds per protocol" | tee -a $LOG_FILE
    echo "Overall throughput: $(echo "scale=2; $TRANSACTION_COUNT / $TOTAL_TIME" | bc -l) TPS" | tee -a $LOG_FILE
else
    echo "❌ TIMEOUT: Convergence not achieved within ${MAX_ROUNDS} rounds" | tee -a $LOG_FILE
fi

echo "" | tee -a $LOG_FILE
echo "Final protocol states:" | tee -a $LOG_FILE
for protocol in pow pos pbft lscc; do
    if [ "${NODE_STATUS[$protocol]}" = "healthy" ]; then
        echo "  [$protocol] ${PROTOCOL_STATES[$protocol]}" | tee -a $LOG_FILE
    fi
done

echo "" | tee -a $LOG_FILE
echo "End time: $(date)" | tee -a $LOG_FILE
echo "Log saved to: $LOG_FILE" | tee -a $LOG_FILE

# Return appropriate exit code
if [ "$CONVERGED" = true ]; then
    exit 0
else
    exit 1
fi