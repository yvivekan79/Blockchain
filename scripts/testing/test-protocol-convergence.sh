#!/bin/bash

# Test Protocol Convergence Script
# Tests that the consensus algorithm converges properly

set -e

SERVER_IP=${1:-"localhost"}
PORT=${2:-5000}
DURATION=${3:-30}

echo "=== Testing Protocol Convergence ==="
echo "Server: $SERVER_IP:$PORT"
echo "Duration: ${DURATION}s"
echo ""

# Check if server is running
echo "Checking server health..."
if ! curl -s --connect-timeout 5 "http://$SERVER_IP:$PORT/health" > /dev/null 2>&1; then
    echo "Server not responding at $SERVER_IP:$PORT"
    exit 1
fi
echo "Server is healthy"
echo ""

# Get initial state
echo "Getting initial blockchain state..."
INITIAL_INFO=$(curl -s "http://$SERVER_IP:$PORT/api/v1/blockchain/info" 2>/dev/null || echo '{}')
INITIAL_HEIGHT=$(echo "$INITIAL_INFO" | jq -r '.chain_height // .height // 0' 2>/dev/null)
INITIAL_TXS=$(echo "$INITIAL_INFO" | jq -r '.total_transactions // .transaction_count // 0' 2>/dev/null)
ALGORITHM=$(echo "$INITIAL_INFO" | jq -r '.consensus_algorithm // "unknown"' 2>/dev/null)

echo "  Algorithm: $ALGORITHM"
echo "  Initial height: $INITIAL_HEIGHT"
echo "  Initial transactions: $INITIAL_TXS"
echo ""

# Inject test transactions
echo "Injecting test transactions..."
INJECT_RESULT=$(curl -s -X POST "http://$SERVER_IP:$PORT/api/v1/transaction-injection/inject-batch" \
    -H "Content-Type: application/json" \
    -d '{"count": 50}' 2>/dev/null || echo '{"error": "failed"}')

INJECTED=$(echo "$INJECT_RESULT" | jq -r '.successful // .injected // 0' 2>/dev/null)
echo "  Injected: $INJECTED transactions"
echo ""

# Monitor convergence
echo "Monitoring convergence for ${DURATION}s..."
START_TIME=$(date +%s)
END_TIME=$((START_TIME + DURATION))

while [ $(date +%s) -lt $END_TIME ]; do
    CURRENT_INFO=$(curl -s "http://$SERVER_IP:$PORT/api/v1/blockchain/info" 2>/dev/null || echo '{}')
    CURRENT_HEIGHT=$(echo "$CURRENT_INFO" | jq -r '.chain_height // .height // 0' 2>/dev/null)
    CURRENT_TXS=$(echo "$CURRENT_INFO" | jq -r '.total_transactions // .transaction_count // 0' 2>/dev/null)
    
    ELAPSED=$(($(date +%s) - START_TIME))
    HEIGHT_DIFF=$((CURRENT_HEIGHT - INITIAL_HEIGHT))
    TXS_DIFF=$((CURRENT_TXS - INITIAL_TXS))
    
    printf "\r  [%3ds] Height: %d (+%d) | Txs: %d (+%d)    " \
        "$ELAPSED" "$CURRENT_HEIGHT" "$HEIGHT_DIFF" "$CURRENT_TXS" "$TXS_DIFF"
    
    sleep 2
done

echo ""
echo ""

# Final state
echo "Getting final blockchain state..."
FINAL_INFO=$(curl -s "http://$SERVER_IP:$PORT/api/v1/blockchain/info" 2>/dev/null || echo '{}')
FINAL_HEIGHT=$(echo "$FINAL_INFO" | jq -r '.chain_height // .height // 0' 2>/dev/null)
FINAL_TXS=$(echo "$FINAL_INFO" | jq -r '.total_transactions // .transaction_count // 0' 2>/dev/null)

HEIGHT_DIFF=$((FINAL_HEIGHT - INITIAL_HEIGHT))
TXS_DIFF=$((FINAL_TXS - INITIAL_TXS))

echo ""
echo "=== Convergence Test Results ==="
echo "Algorithm: $ALGORITHM"
echo "Duration: ${DURATION}s"
echo "Height change: $INITIAL_HEIGHT -> $FINAL_HEIGHT (+$HEIGHT_DIFF blocks)"
echo "Transaction change: $INITIAL_TXS -> $FINAL_TXS (+$TXS_DIFF)"
echo ""

# Evaluate results
if [ $HEIGHT_DIFF -gt 0 ]; then
    echo "Block processing: WORKING"
    BLOCKS_PER_SEC=$(echo "scale=2; $HEIGHT_DIFF / $DURATION" | bc -l 2>/dev/null || echo "N/A")
    echo "Block rate: $BLOCKS_PER_SEC blocks/sec"
else
    echo "Block processing: NO NEW BLOCKS"
fi

if [ $TXS_DIFF -gt 0 ]; then
    echo "Transaction processing: WORKING"
    TPS=$(echo "scale=2; $TXS_DIFF / $DURATION" | bc -l 2>/dev/null || echo "N/A")
    echo "TPS: $TPS"
else
    echo "Transaction processing: NO NEW TRANSACTIONS"
fi

echo ""
if [ $HEIGHT_DIFF -gt 0 ] && [ $TXS_DIFF -gt 0 ]; then
    echo "CONVERGENCE TEST: PASSED"
    exit 0
else
    echo "CONVERGENCE TEST: NEEDS INVESTIGATION"
    exit 1
fi
