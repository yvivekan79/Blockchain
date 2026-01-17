
#!/bin/bash

# Convergence Benchmark Test - 150 transactions per protocol per node
# Tests convergence time across 4 nodes × 4 algorithms = 16 endpoints

set -e

# Node and protocol configuration
NODES=("192.168.50.147" "192.168.50.148" "192.168.50.149" "192.168.50.150")
ALGORITHMS=("pow" "pos" "pbft" "lscc")
PORTS=(5001 5002 5003 5004)
TRANSACTIONS_PER_ENDPOINT=150

TEST_ID="convergence_benchmark_$(date +%Y%m%d_%H%M%S)"
LOG_FILE="${TEST_ID}.log"
RESULTS_FILE="${TEST_ID}_results.json"

echo "=== Convergence Benchmark Test ===" | tee $LOG_FILE
echo "Test ID: $TEST_ID" | tee -a $LOG_FILE
echo "Configuration: ${TRANSACTIONS_PER_ENDPOINT} transactions per endpoint" | tee -a $LOG_FILE
echo "Total endpoints: $(( ${#NODES[@]} * ${#ALGORITHMS[@]} ))" | tee -a $LOG_FILE
echo "Total transactions: $(( ${#NODES[@]} * ${#ALGORITHMS[@]} * TRANSACTIONS_PER_ENDPOINT ))" | tee -a $LOG_FILE
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

# Function to get endpoint URL
get_endpoint() {
    local node_ip=$1
    local port=$2
    echo "http://${node_ip}:${port}"
}

# Function to check endpoint health
check_endpoint_health() {
    local endpoint=$1
    local response=$(curl -s --connect-timeout 5 --max-time 10 "${endpoint}/health" 2>/dev/null || echo "")
    
    if [[ "$response" == *"healthy"* ]] || [[ "$response" == *"ok"* ]] || [[ "$response" == *"status"* ]]; then
        return 0
    else
        return 1
    fi
}

# Function to get initial blockchain state
get_initial_state() {
    local endpoint=$1
    local response=$(curl -s --connect-timeout 10 --max-time 15 "${endpoint}/api/v1/blockchain/info" 2>/dev/null || echo '{"error": "no_response"}')
    
    # Validate JSON response
    if echo "$response" | jq empty 2>/dev/null; then
        echo "$response"
    else
        echo '{"error": "invalid_json", "raw_response": "'"${response//\"/\\\"}"'"}'
    fi
}

# Function to inject transactions with better error handling
inject_transactions() {
    local endpoint=$1
    local algorithm=$2
    local node_num=$3
    
    echo "  Injecting ${TRANSACTIONS_PER_ENDPOINT} transactions to Node${node_num} ${algorithm^^}..." | tee -a $LOG_FILE
    
    # Create proper JSON payload
    local payload='{"count": '${TRANSACTIONS_PER_ENDPOINT}', "rate_per_second": 50}'
    
    local response=$(curl -s --connect-timeout 15 --max-time 30 \
        -X POST "${endpoint}/api/v1/transaction-injection/inject-batch" \
        -H "Content-Type: application/json" \
        -d "$payload" 2>/dev/null || echo '{"error": "injection_failed"}')
    
    # Validate and parse response
    if echo "$response" | jq empty 2>/dev/null; then
        local successful=$(extract_json_value "$response" ".successful" "0")
        local duration=$(extract_json_value "$response" ".duration_ms" "0")
        local tps=$(extract_json_value "$response" ".actual_tps" "0")
        
        echo "    Result: ${successful}/${TRANSACTIONS_PER_ENDPOINT} successful, ${duration}ms, ${tps} TPS" | tee -a $LOG_FILE
        echo "$response"
    else
        echo "    Result: Injection failed - invalid response" | tee -a $LOG_FILE
        echo '{"error": "invalid_response", "successful": 0, "duration_ms": 0, "actual_tps": 0}'
    fi
}

# Function to monitor convergence
monitor_convergence() {
    local endpoint=$1
    local algorithm=$2
    local node_num=$3
    
    local info_response=$(curl -s --connect-timeout 10 --max-time 15 "${endpoint}/api/v1/blockchain/info" 2>/dev/null || echo '{}')
    local consensus_response=$(curl -s --connect-timeout 10 --max-time 15 "${endpoint}/api/v1/consensus/status" 2>/dev/null || echo '{}')
    
    # Extract values with safe defaults
    local height=0
    local transactions=0
    local status="unknown"
    
    if echo "$info_response" | jq empty 2>/dev/null; then
        height=$(extract_json_value "$info_response" ".chain_height" "0")
        transactions=$(extract_json_value "$info_response" ".total_transactions" "0")
    fi
    
    if echo "$consensus_response" | jq empty 2>/dev/null; then
        status=$(extract_json_value "$consensus_response" ".status" "unknown")
    fi
    
    printf "Node%d %-4s | Height: %-4s | Txs: %-4s | Status: %-10s\n" \
           "$node_num" "${algorithm^^}" "$height" "$transactions" "$status"
    
    echo "{\"node\": $node_num, \"algorithm\": \"$algorithm\", \"height\": $height, \"transactions\": $transactions, \"status\": \"$status\"}"
}

# Phase 1: Health check all endpoints
echo "Phase 1: Health Check" | tee -a $LOG_FILE
echo "Checking all 16 endpoints..." | tee -a $LOG_FILE

healthy_endpoints=0
declare -A endpoint_status

for i in "${!NODES[@]}"; do
    node_ip=${NODES[$i]}
    node_num=$((i + 1))
    
    for j in "${!ALGORITHMS[@]}"; do
        algorithm=${ALGORITHMS[$j]}
        port=${PORTS[$j]}
        endpoint=$(get_endpoint $node_ip $port)
        
        if check_endpoint_health "$endpoint"; then
            echo "✅ Node${node_num} ${algorithm^^} (${node_ip}:${port}) - Healthy" | tee -a $LOG_FILE
            endpoint_status["${node_num}_${algorithm}"]="healthy"
            healthy_endpoints=$((healthy_endpoints + 1))
        else
            echo "❌ Node${node_num} ${algorithm^^} (${node_ip}:${port}) - Down" | tee -a $LOG_FILE
            endpoint_status["${node_num}_${algorithm}"]="down"
        fi
    done
done

echo "" | tee -a $LOG_FILE
echo "Healthy endpoints: ${healthy_endpoints}/16" | tee -a $LOG_FILE

if [ $healthy_endpoints -lt 4 ]; then
    echo "❌ ERROR: Too few healthy endpoints for meaningful convergence test" | tee -a $LOG_FILE
    exit 1
fi

# Phase 2: Get initial states
echo "" | tee -a $LOG_FILE
echo "Phase 2: Recording Initial States" | tee -a $LOG_FILE

# Create results file with proper JSON structure
cat > $RESULTS_FILE << EOF
{
  "test_info": {
    "test_id": "$TEST_ID",
    "start_time": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "transactions_per_endpoint": $TRANSACTIONS_PER_ENDPOINT,
    "healthy_endpoints": $healthy_endpoints
  },
  "initial_states": {
EOF

first_state=true
for i in "${!NODES[@]}"; do
    node_ip=${NODES[$i]}
    node_num=$((i + 1))
    
    for j in "${!ALGORITHMS[@]}"; do
        algorithm=${ALGORITHMS[$j]}
        port=${PORTS[$j]}
        endpoint=$(get_endpoint $node_ip $port)
        
        if [ "${endpoint_status["${node_num}_${algorithm}"]}" = "healthy" ]; then
            if [ "$first_state" = false ]; then
                echo "," >> $RESULTS_FILE
            fi
            first_state=false
            
            initial_state=$(get_initial_state "$endpoint")
            echo "    \"node${node_num}_${algorithm}\": $initial_state" >> $RESULTS_FILE
        fi
    done
done

echo "  }," >> $RESULTS_FILE

# Phase 3: Simultaneous transaction injection
echo "" | tee -a $LOG_FILE
echo "Phase 3: Simultaneous Transaction Injection" | tee -a $LOG_FILE
echo "Injecting ${TRANSACTIONS_PER_ENDPOINT} transactions to each healthy endpoint..." | tee -a $LOG_FILE

injection_start_time=$(date +%s)

echo "  \"injection_results\": {" >> $RESULTS_FILE
first_injection=true

# Inject to all endpoints simultaneously using background processes
pids=()
injection_files=()

for i in "${!NODES[@]}"; do
    node_ip=${NODES[$i]}
    node_num=$((i + 1))
    
    for j in "${!ALGORITHMS[@]}"; do
        algorithm=${ALGORITHMS[$j]}
        port=${PORTS[$j]}
        endpoint=$(get_endpoint $node_ip $port)
        
        if [ "${endpoint_status["${node_num}_${algorithm}"]}" = "healthy" ]; then
            injection_file="/tmp/injection_${node_num}_${algorithm}.json"
            injection_files+=("$injection_file")
            
            # Background injection
            (
                result=$(inject_transactions "$endpoint" "$algorithm" "$node_num")
                echo "$result" > "$injection_file"
            ) &
            pids+=($!)
        fi
    done
done

# Wait for all injections to complete
echo "Waiting for all injections to complete..." | tee -a $LOG_FILE
for pid in "${pids[@]}"; do
    wait $pid 2>/dev/null || true
done

injection_end_time=$(date +%s)
injection_duration=$((injection_end_time - injection_start_time))

echo "All injections completed in ${injection_duration}s" | tee -a $LOG_FILE

# Collect injection results
for i in "${!NODES[@]}"; do
    node_ip=${NODES[$i]}
    node_num=$((i + 1))
    
    for j in "${!ALGORITHMS[@]}"; do
        algorithm=${ALGORITHMS[$j]}
        
        if [ "${endpoint_status["${node_num}_${algorithm}"]}" = "healthy" ]; then
            injection_file="/tmp/injection_${node_num}_${algorithm}.json"
            
            if [ "$first_injection" = false ]; then
                echo "," >> $RESULTS_FILE
            fi
            first_injection=false
            
            if [ -f "$injection_file" ]; then
                injection_result=$(cat "$injection_file")
                echo "    \"node${node_num}_${algorithm}\": $injection_result" >> $RESULTS_FILE
                rm "$injection_file" 2>/dev/null || true
            else
                echo "    \"node${node_num}_${algorithm}\": {\"error\": \"no_result_file\"}" >> $RESULTS_FILE
            fi
        fi
    done
done

echo "  }," >> $RESULTS_FILE

# Phase 4: Convergence monitoring
echo "" | tee -a $LOG_FILE
echo "Phase 4: Convergence Monitoring" | tee -a $LOG_FILE
echo "Monitoring convergence every 10 seconds for up to 3 minutes..." | tee -a $LOG_FILE
echo "" | tee -a $LOG_FILE

convergence_start_time=$(date +%s)
max_monitoring_time=180  # 3 minutes
monitoring_interval=10

echo "  \"convergence_monitoring\": [" >> $RESULTS_FILE

convergence_round=0
converged=false

while [ $(($(date +%s) - convergence_start_time)) -lt $max_monitoring_time ] && [ "$converged" = false ]; do
    convergence_round=$((convergence_round + 1))
    round_time=$(date +%s)
    
    echo "=== Convergence Round $convergence_round ===" | tee -a $LOG_FILE
    
    if [ $convergence_round -gt 1 ]; then
        echo "," >> $RESULTS_FILE
    fi
    
    echo "    {" >> $RESULTS_FILE
    echo "      \"round\": $convergence_round," >> $RESULTS_FILE
    echo "      \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"," >> $RESULTS_FILE
    echo "      \"elapsed_seconds\": $((round_time - convergence_start_time))," >> $RESULTS_FILE
    echo "      \"states\": {" >> $RESULTS_FILE
    
    first_monitor=true
    heights=()
    transactions=()
    
    for i in "${!NODES[@]}"; do
        node_ip=${NODES[$i]}
        node_num=$((i + 1))
        
        for j in "${!ALGORITHMS[@]}"; do
            algorithm=${ALGORITHMS[$j]}
            port=${PORTS[$j]}
            endpoint=$(get_endpoint $node_ip $port)
            
            if [ "${endpoint_status["${node_num}_${algorithm}"]}" = "healthy" ]; then
                if [ "$first_monitor" = false ]; then
                    echo "," >> $RESULTS_FILE
                fi
                first_monitor=false
                
                state=$(monitor_convergence "$endpoint" "$algorithm" "$node_num")
                echo "        \"node${node_num}_${algorithm}\": $state" >> $RESULTS_FILE
                
                # Extract metrics for convergence analysis
                height=$(extract_json_value "$state" ".height" "0")
                tx_count=$(extract_json_value "$state" ".transactions" "0")
                heights+=($height)
                transactions+=($tx_count)
            fi
        done
    done
    
    echo "      }" >> $RESULTS_FILE
    echo "    }" >> $RESULTS_FILE
    
    # Analyze convergence per protocol
    declare -A protocol_heights
    declare -A protocol_transactions
    declare -A protocol_convergence_status
    
    # Group data by protocol
    for i in "${!NODES[@]}"; do
        node_ip=${NODES[$i]}
        node_num=$((i + 1))
        
        for j in "${!ALGORITHMS[@]}"; do
            algorithm=${ALGORITHMS[$j]}
            port=${PORTS[$j]}
            endpoint=$(get_endpoint $node_ip $port)
            
            if [ "${endpoint_status["${node_num}_${algorithm}"]}" = "healthy" ]; then
                state=$(monitor_convergence "$endpoint" "$algorithm" "$node_num")
                height=$(extract_json_value "$state" ".height" "0")
                tx_count=$(extract_json_value "$state" ".transactions" "0")
                
                if [ -z "${protocol_heights[$algorithm]}" ]; then
                    protocol_heights[$algorithm]="$height"
                    protocol_transactions[$algorithm]="$tx_count"
                else
                    protocol_heights[$algorithm]="${protocol_heights[$algorithm]},$height"
                    protocol_transactions[$algorithm]="${protocol_transactions[$algorithm]},$tx_count"
                fi
            fi
        done
    done
    
    # Analyze per-protocol convergence
    echo "Per-Protocol Analysis:" | tee -a $LOG_FILE
    protocol_converged_count=0
    
    for algorithm in "${ALGORITHMS[@]}"; do
        if [ -n "${protocol_heights[$algorithm]}" ]; then
            # Calculate variance for this protocol
            IFS=',' read -ra heights_array <<< "${protocol_heights[$algorithm]}"
            IFS=',' read -ra tx_array <<< "${protocol_transactions[$algorithm]}"
            
            if [ ${#heights_array[@]} -gt 0 ]; then
                min_height=$(printf '%s\n' "${heights_array[@]}" | sort -n | head -n1)
                max_height=$(printf '%s\n' "${heights_array[@]}" | sort -n | tail -n1)
                height_variance=$((max_height - min_height))
                
                min_tx=$(printf '%s\n' "${tx_array[@]}" | sort -n | head -n1)
                max_tx=$(printf '%s\n' "${tx_array[@]}" | sort -n | tail -n1)
                tx_variance=$((max_tx - min_tx))
                
                # Calculate average values
                total_height=0
                for h in "${heights_array[@]}"; do
                    total_height=$((total_height + h))
                done
                avg_height=$((total_height / ${#heights_array[@]}))
                
                total_tx=0
                for t in "${tx_array[@]}"; do
                    total_tx=$((total_tx + t))
                done
                avg_tx=$((total_tx / ${#tx_array[@]}))
                
                # Check protocol-specific convergence
                if [ $height_variance -le 3 ] && [ $tx_variance -le 50 ] && [ $avg_tx -gt 50 ]; then
                    protocol_convergence_status[$algorithm]="CONVERGED"
                    protocol_converged_count=$((protocol_converged_count + 1))
                    convergence_indicator="✅"
                elif [ $avg_tx -gt 20 ]; then
                    protocol_convergence_status[$algorithm]="CONVERGING"
                    convergence_indicator="⏳"
                else
                    protocol_convergence_status[$algorithm]="SLOW"
                    convergence_indicator="⚠️"
                fi
                
                echo "  ${convergence_indicator} ${algorithm^^}: Height ${min_height}-${max_height} (var:${height_variance}), Tx ${min_tx}-${max_tx} (var:${tx_variance}), Avg: ${avg_height}h/${avg_tx}tx" | tee -a $LOG_FILE
            fi
        else
            protocol_convergence_status[$algorithm]="NO_DATA"
            echo "  ❌ ${algorithm^^}: No data available" | tee -a $LOG_FILE
        fi
    done
    
    # Overall convergence decision
    if [ $protocol_converged_count -ge 3 ]; then
        converged=true
        convergence_time=$(($(date +%s) - convergence_start_time))
        echo "✅ MULTI-PROTOCOL CONVERGENCE ACHIEVED in ${convergence_time}s (${protocol_converged_count}/${#ALGORITHMS[@]} protocols)" | tee -a $LOG_FILE
    elif [ $protocol_converged_count -ge 1 ]; then
        echo "⏳ Partial convergence: ${protocol_converged_count}/${#ALGORITHMS[@]} protocols converged" | tee -a $LOG_FILE
    else
        echo "⚠️  No protocols have achieved convergence yet" | tee -a $LOG_FILE
    fi
    
    echo "" | tee -a $LOG_FILE
    
    if [ "$converged" = false ]; then
        sleep $monitoring_interval
    fi
done

echo "  ]," >> $RESULTS_FILE

# Final results
final_time=$(date +%s)
total_test_time=$((final_time - injection_start_time))

echo "  \"final_results\": {" >> $RESULTS_FILE
echo "    \"converged\": $([ "$converged" = true ] && echo 'true' || echo 'false')," >> $RESULTS_FILE
echo "    \"total_test_time_seconds\": $total_test_time," >> $RESULTS_FILE
echo "    \"convergence_rounds\": $convergence_round," >> $RESULTS_FILE
echo "    \"protocol_convergence_status\": {" >> $RESULTS_FILE

first_protocol=true
for algorithm in "${ALGORITHMS[@]}"; do
    if [ "$first_protocol" = false ]; then
        echo "," >> $RESULTS_FILE
    fi
    first_protocol=false
    
    status="${protocol_convergence_status[$algorithm]:-NO_DATA}"
    echo "      \"$algorithm\": \"$status\"" >> $RESULTS_FILE
done

echo "    }" >> $RESULTS_FILE

if [ "$converged" = true ]; then
    echo "    ,\"convergence_time_seconds\": $convergence_time" >> $RESULTS_FILE
fi

echo "  }" >> $RESULTS_FILE
echo "}" >> $RESULTS_FILE

# Summary
echo "" | tee -a $LOG_FILE
echo "=== Test Summary ===" | tee -a $LOG_FILE
echo "Test ID: $TEST_ID" | tee -a $LOG_FILE
echo "Total endpoints tested: $healthy_endpoints" | tee -a $LOG_FILE
echo "Transactions per endpoint: $TRANSACTIONS_PER_ENDPOINT" | tee -a $LOG_FILE
echo "Total test time: ${total_test_time}s" | tee -a $LOG_FILE

echo "" | tee -a $LOG_FILE
echo "Protocol Performance Summary:" | tee -a $LOG_FILE
for algorithm in "${ALGORITHMS[@]}"; do
    status="${protocol_convergence_status[$algorithm]:-NO_DATA}"
    case $status in
        "CONVERGED")
            echo "✅ ${algorithm^^}: Converged successfully" | tee -a $LOG_FILE
            ;;
        "CONVERGING")
            echo "⏳ ${algorithm^^}: Still converging" | tee -a $LOG_FILE
            ;;
        "SLOW")
            echo "⚠️  ${algorithm^^}: Slow convergence" | tee -a $LOG_FILE
            ;;
        *)
            echo "❌ ${algorithm^^}: No convergence data" | tee -a $LOG_FILE
            ;;
    esac
done

if [ "$converged" = true ]; then
    echo "" | tee -a $LOG_FILE
    echo "✅ Multi-protocol convergence achieved in: ${convergence_time}s" | tee -a $LOG_FILE
    echo "   Converged protocols: ${protocol_converged_count}/${#ALGORITHMS[@]}" | tee -a $LOG_FILE
else
    echo "" | tee -a $LOG_FILE
    echo "❌ Full convergence not achieved within monitoring period" | tee -a $LOG_FILE
    echo "   Partial convergence: ${protocol_converged_count}/${#ALGORITHMS[@]} protocols" | tee -a $LOG_FILE
fi

echo "Results saved to: $RESULTS_FILE" | tee -a $LOG_FILE
echo "Log saved to: $LOG_FILE" | tee -a $LOG_FILE

echo "" | tee -a $LOG_FILE
echo "🎯 CONVERGENCE TEST COMPLETED" | tee -a $LOG_FILE
