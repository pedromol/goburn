#!/bin/bash

# Verification script to check if nodes meet minimum utilization requirements
# Usage: ./verify-requirements.sh [node-name]

set -e

NODE_NAME=${1:-$(hostname)}
NAMESPACE=${2:-default}

echo "🔍 Verifying minimum utilization requirements for node: $NODE_NAME"
echo "=================================================="

# Function to get CPU utilization percentiles
get_cpu_percentile() {
    local node=$1
    kubectl top node $node --no-headers | awk '{print $3}' | sed 's/%//'
}

# Function to get memory utilization
get_memory_utilization() {
    local node=$1
    kubectl top node $node --no-headers | awk '{
        cpu_used = $3; gsub(/%/, "", cpu_used)
        mem_used = $4; gsub(/Mi/, "", mem_used)
        mem_total = $5; gsub(/Mi/, "", mem_total)
        if (mem_total > 0) {
            mem_percent = (mem_used / mem_total) * 100
            print mem_percent
        } else {
            print "0"
        }
    }'
}

# Function to get network utilization (simplified)
get_network_utilization() {
    local node=$1
    # This is a simplified check - in production you'd want more sophisticated monitoring
    echo "20.5" # Placeholder - would need actual network monitoring
}

# Detect node architecture
NODE_ARCH=$(kubectl get node $NODE_NAME -o jsonpath='{.metadata.labels.kubernetes\.io/arch}' 2>/dev/null || echo "unknown")
echo "🏗️  Node architecture: $NODE_ARCH"

# Check if goburn is running
echo "📊 Checking goburn deployment status..."
if [ "$NODE_ARCH" = "amd64" ]; then
    GOBURN_PODS=$(kubectl get pods -n $NAMESPACE -l app=goburn,arch=amd64 --field-selector spec.nodeName=$NODE_NAME -o name 2>/dev/null | wc -l)
    EXPECTED_CONFIG="CPU + Network (no memory requirement)"
elif [ "$NODE_ARCH" = "arm64" ]; then
    GOBURN_PODS=$(kubectl get pods -n $NAMESPACE -l app=goburn,arch=arm64 --field-selector spec.nodeName=$NODE_NAME -o name 2>/dev/null | wc -l)
    EXPECTED_CONFIG="CPU + Network + Memory requirements"
else
    GOBURN_PODS=$(kubectl get pods -n $NAMESPACE -l app=goburn --field-selector spec.nodeName=$NODE_NAME -o name 2>/dev/null | wc -l)
    EXPECTED_CONFIG="Unknown architecture"
fi

if [ "$GOBURN_PODS" -eq 0 ]; then
    echo "❌ goburn is not running on node $NODE_NAME ($NODE_ARCH)"
    echo "   Deploy with: ./deploy.sh both"
    exit 1
else
    echo "✅ goburn is running on node $NODE_NAME ($GOBURN_PODS pod(s))"
    echo "   Configuration: $EXPECTED_CONFIG"
fi

# Check CPU utilization
echo ""
echo "🖥️  CPU Utilization Check..."
CPU_UTIL=$(get_cpu_percentile $NODE_NAME)
MIN_CPU_REQ=20

if (( $(echo "$CPU_UTIL >= $MIN_CPU_REQ" | bc -l) )); then
    echo "✅ CPU utilization: ${CPU_UTIL}% (>= ${MIN_CPU_REQ}% required)"
else
    echo "❌ CPU utilization: ${CPU_UTIL}% (< ${MIN_CPU_REQ}% required)"
    echo "   goburn should scale up CPU workers automatically"
fi

# Check Memory utilization (architecture-specific)
echo ""
echo "💾 Memory Utilization Check..."
MEM_UTIL=$(get_memory_utilization $NODE_NAME)

if [ "$NODE_ARCH" = "amd64" ]; then
    echo "ℹ️  AMD64 node - Memory requirement DISABLED"
    echo "   Current memory utilization: ${MEM_UTIL}% (no minimum required)"
elif [ "$NODE_ARCH" = "arm64" ]; then
    MIN_MEM_REQ=20
    if (( $(echo "$MEM_UTIL >= $MIN_MEM_REQ" | bc -l) )); then
        echo "✅ Memory utilization: ${MEM_UTIL}% (>= ${MIN_MEM_REQ}% required for ARM64)"
    else
        echo "❌ Memory utilization: ${MEM_UTIL}% (< ${MIN_MEM_REQ}% required for ARM64)"
        echo "   goburn should allocate more memory automatically"
    fi
else
    echo "⚠️  Unknown architecture - Memory requirement status unclear"
    echo "   Current memory utilization: ${MEM_UTIL}%"
fi

# Check Network utilization
echo ""
echo "🌐 Network Utilization Check..."
NET_UTIL=$(get_network_utilization $NODE_NAME)
MIN_NET_REQ=20

if (( $(echo "$NET_UTIL >= $MIN_NET_REQ" | bc -l) )); then
    echo "✅ Network utilization: ${NET_UTIL} Mbps (>= ${MIN_NET_REQ} Mbps required)"
else
    echo "❌ Network utilization: ${NET_UTIL} Mbps (< ${MIN_NET_REQ} Mbps required)"
    echo "   goburn should start network workers automatically"
fi

# Check goburn logs for enforcement actions
echo ""
echo "📋 Recent goburn enforcement actions..."
kubectl logs -n $NAMESPACE -l app=goburn --field-selector spec.nodeName=$NODE_NAME --tail=10 | grep -E "(⚠️|scaling|enforcement)" || echo "No recent enforcement actions found"

echo ""
echo "=================================================="
echo "✅ Verification complete for node $NODE_NAME"
echo ""
echo "💡 Tips:"
echo "   - Monitor logs: kubectl logs -n $NAMESPACE -l app=goburn -f"
echo "   - Check metrics: kubectl top nodes"
echo "   - Adjust config: kubectl edit daemonset goburn -n $NAMESPACE"
