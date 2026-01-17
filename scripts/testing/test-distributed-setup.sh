#!/bin/bash

# Test script to verify distributed LSCC blockchain deployment
# This tests a single-protocol cluster across multiple nodes

echo "LSCC Blockchain Distributed Setup Verification"
echo "==============================================="

# Default nodes
NODES=("192.168.50.147" "192.168.50.148" "192.168.50.149" "192.168.50.150")
PORT=5000

# Check command line argument for custom nodes
if [ -n "$1" ]; then
    NODES=("$@")
fi

echo "Testing nodes: ${NODES[*]}"
echo "Port: $PORT"
echo ""

# Phase 1: Health Check
echo "Phase 1: Health Check"
echo "====================="

active_nodes=0
for node in "${NODES[@]}"; do
    echo -n "  Node $node: "
    if curl -s --connect-timeout 3 "http://$node:$PORT/health" > /dev/null 2>&1; then
        echo "HEALTHY"
        ((active_nodes++))
    else
        echo "NOT RESPONDING"
    fi
done

echo ""
echo "Active nodes: $active_nodes/${#NODES[@]}"

if [ $active_nodes -eq 0 ]; then
    echo "No nodes are responding. Please start the cluster first."
    echo "Use: ./scripts/deployment/deploy-cluster.sh start"
    exit 1
fi

# Phase 2: Network Connectivity
echo ""
echo "Phase 2: Network Connectivity"
echo "=============================="

for node in "${NODES[@]}"; do
    peers=$(curl -s --connect-timeout 3 "http://$node:$PORT/api/v1/network/peers" 2>/dev/null)
    if [ -n "$peers" ]; then
        peer_count=$(echo "$peers" | jq -r '.peer_count // .connected_peers // 0' 2>/dev/null)
        echo "  Node $node: $peer_count peers connected"
    fi
done

# Phase 3: Blockchain State
echo ""
echo "Phase 3: Blockchain State"
echo "========================="

for node in "${NODES[@]}"; do
    info=$(curl -s --connect-timeout 3 "http://$node:$PORT/api/v1/blockchain/info" 2>/dev/null)
    if [ -n "$info" ]; then
        height=$(echo "$info" | jq -r '.chain_height // .height // 0' 2>/dev/null)
        txs=$(echo "$info" | jq -r '.total_transactions // .transaction_count // 0' 2>/dev/null)
        algo=$(echo "$info" | jq -r '.consensus_algorithm // "unknown"' 2>/dev/null)
        echo "  Node $node: Height=$height, Txs=$txs, Algorithm=$algo"
    fi
done

# Phase 4: Shard Status
echo ""
echo "Phase 4: Shard Status"
echo "====================="

for node in "${NODES[@]}"; do
    shards=$(curl -s --connect-timeout 3 "http://$node:$PORT/api/v1/shards" 2>/dev/null)
    if [ -n "$shards" ]; then
        active=$(echo "$shards" | jq -r '.active_shards // 0' 2>/dev/null)
        total=$(echo "$shards" | jq -r '.total_shards // 0' 2>/dev/null)
        echo "  Node $node: $active/$total shards active"
    fi
done

# Summary
echo ""
echo "VERIFICATION SUMMARY"
echo "===================="
echo "Active nodes: $active_nodes/${#NODES[@]}"

if [ $active_nodes -ge 2 ]; then
    echo "Distributed cluster is OPERATIONAL"
else
    echo "Cluster needs more nodes for distributed operation"
fi

echo ""
echo "Network Verification Commands:"
echo "=============================="
echo "# Check network status:"
echo "curl http://NODE_IP:$PORT/api/v1/network/status"
echo ""
echo "# Check connected peers:"
echo "curl http://NODE_IP:$PORT/api/v1/network/peers"
echo ""
echo "# Check blockchain info:"
echo "curl http://NODE_IP:$PORT/api/v1/blockchain/info"

exit 0
