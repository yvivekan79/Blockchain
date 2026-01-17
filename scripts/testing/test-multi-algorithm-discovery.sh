
#!/bin/bash

echo "🧪 Testing Multi-Algorithm P2P Network Discovery"
echo "==============================================="

# Test each algorithm's network status
test_algorithm_discovery() {
    local algorithm=$1
    local port=$2
    local expected_p2p_port=$3
    
    echo ""
    echo "🔍 Testing $algorithm algorithm (Port: $port)..."
    echo "Expected P2P Port: $expected_p2p_port"
    echo "Expected Consensus: $algorithm"
    echo "---"
    
    response=$(curl -s http://192.168.50.147:$port/api/v1/network/status)
    
    if [ $? -eq 0 ]; then
        echo "✅ Server responsive"
        
        # Extract values using jq if available, otherwise grep
        if command -v jq >/dev/null 2>&1; then
            consensus=$(echo "$response" | jq -r '.distributed_network.node_info.consensus_algorithm')
            listen_port=$(echo "$response" | jq -r '.distributed_network.node_info.listen_port')
            node_id=$(echo "$response" | jq -r '.distributed_network.node_info.id')
            
            echo "📊 Results:"
            echo "   Node ID: $node_id"
            echo "   Consensus Algorithm: $consensus"
            echo "   Listen Port: $listen_port"
            
            # Verify correctness
            if [ "$consensus" = "$algorithm" ]; then
                echo "   ✅ Consensus algorithm matches expected: $algorithm"
            else
                echo "   ❌ Consensus algorithm mismatch. Expected: $algorithm, Got: $consensus"
            fi
            
            if [ "$listen_port" = "$expected_p2p_port" ]; then
                echo "   ✅ Listen port matches expected: $expected_p2p_port"
            else
                echo "   ❌ Listen port mismatch. Expected: $expected_p2p_port, Got: $listen_port"
            fi
        else
            echo "📄 Raw response (install jq for parsed output):"
            echo "$response" | head -10
        fi
    else
        echo "❌ Server not responsive"
    fi
}

# Test all algorithms
test_algorithm_discovery "pow" 5001 9001
test_algorithm_discovery "pos" 5002 9002
test_algorithm_discovery "pbft" 5003 9003
test_algorithm_discovery "lscc" 5004 9004

echo ""
echo "🎯 P2P Discovery Test Complete"
echo "============================="
echo ""
echo "📋 Summary:"
echo "- Each algorithm should have its own consensus type"
echo "- Each algorithm should have its own P2P port (9001-9004)"
echo "- All nodes should share the same node ID but different algorithms"
echo ""
echo "🔧 If issues persist, restart servers with:"
echo "   ./start-multi-algorithm-servers.sh"
