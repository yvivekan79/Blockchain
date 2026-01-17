#!/bin/bash

# Stop all distributed nodes

echo "Stopping all distributed LSCC nodes..."

# Kill all lscc-blockchain processes
pkill -f "lscc-blockchain" 2>/dev/null || true

# Wait for processes to stop
sleep 2

# Check if any are still running
REMAINING=$(ps aux | grep "lscc-blockchain" | grep -v grep | wc -l)

if [ $REMAINING -eq 0 ]; then
    echo "✅ All nodes stopped successfully"
else
    echo "⚠️ Some processes may still be running: $REMAINING"
    echo "Force killing remaining processes..."
    pkill -9 -f "lscc-blockchain" 2>/dev/null || true
fi

echo "Distributed network stopped."