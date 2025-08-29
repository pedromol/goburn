#!/bin/bash

# Deployment script for goburn with architecture-specific configurations
# Usage: ./deploy.sh [amd64|arm64|both|status|cleanup]

set -e

NAMESPACE=${NAMESPACE:-default}

show_help() {
    echo "🔥 goburn Deployment Script"
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  amd64     Deploy only AMD64 DaemonSet (CPU + Network, no memory)"
    echo "  arm64     Deploy only ARM64 DaemonSet (CPU + Network + Memory)"
    echo "  both      Deploy both DaemonSets (default)"
    echo "  status    Show deployment status"
    echo "  cleanup   Remove all goburn resources"
    echo "  help      Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  NAMESPACE     Kubernetes namespace (default: default)"
    echo ""
    echo "Examples:"
    echo "  $0 both              # Deploy to all architectures"
    echo "  $0 amd64             # Deploy only to AMD64 nodes"
    echo "  NAMESPACE=prod $0 both  # Deploy to 'prod' namespace"
}

deploy_rbac() {
    echo "📋 Deploying RBAC resources..."
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: goburn
  namespace: $NAMESPACE
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: goburn
rules:
- apiGroups: [""]
  resources: ["nodes"]
  verbs: ["get", "list"]
- apiGroups: ["metrics.k8s.io"]
  resources: ["nodes", "pods"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: goburn
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: goburn
subjects:
- kind: ServiceAccount
  name: goburn
  namespace: $NAMESPACE
EOF
}

deploy_amd64() {
    echo "🖥️  Deploying AMD64 DaemonSet (CPU + Network only)..."
    kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: goburn-amd64
  namespace: $NAMESPACE
  labels:
    app: goburn
    arch: amd64
spec:
  selector:
    matchLabels:
      app: goburn
      arch: amd64
  template:
    metadata:
      labels:
        app: goburn
        arch: amd64
    spec:
      serviceAccountName: goburn
      hostNetwork: true
      hostPID: true
      containers:
      - name: goburn
        image: pedromol/goburn:latest
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: MIN_CPU_UTILIZATION
          value: "20"
        - name: MIN_MEMORY_UTILIZATION
          value: "0"
        - name: MIN_NETWORK_UTILIZATION_MBPS
          value: "20"
        - name: ENABLE_MEMORY_UTILIZATION
          value: "false"
        - name: TARGET_CPU_UTILIZATION
          value: "80"
        - name: TARGET_MEMORY_UTILIZATION
          value: "80"
        - name: MONITOR_INTERVAL_SECONDS
          value: "30"
        - name: SCALE_UP_DELAY_SECONDS
          value: "60"
        - name: SCALE_DOWN_DELAY_SECONDS
          value: "120"
        - name: MAX_MEMORY_MB
          value: "2048"
        - name: NETWORK_INTERFACE
          value: "eth0"
        resources:
          requests:
            memory: "100Mi"
            cpu: "50m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        securityContext:
          privileged: false
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 1000
          capabilities:
            drop:
            - ALL
      tolerations:
      - operator: Exists
        effect: NoSchedule
      - operator: Exists
        effect: NoExecute
      nodeSelector:
        kubernetes.io/os: linux
        kubernetes.io/arch: amd64
      priorityClassName: system-node-critical
EOF
}

deploy_arm64() {
    echo "💪 Deploying ARM64 DaemonSet (CPU + Network + Memory)..."
    kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: goburn-arm64
  namespace: $NAMESPACE
  labels:
    app: goburn
    arch: arm64
spec:
  selector:
    matchLabels:
      app: goburn
      arch: arm64
  template:
    metadata:
      labels:
        app: goburn
        arch: arm64
    spec:
      serviceAccountName: goburn
      hostNetwork: true
      hostPID: true
      containers:
      - name: goburn
        image: pedromol/goburn:latest
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: MIN_CPU_UTILIZATION
          value: "20"
        - name: MIN_MEMORY_UTILIZATION
          value: "20"
        - name: MIN_NETWORK_UTILIZATION_MBPS
          value: "20"
        - name: ENABLE_MEMORY_UTILIZATION
          value: "true"
        - name: TARGET_CPU_UTILIZATION
          value: "80"
        - name: TARGET_MEMORY_UTILIZATION
          value: "80"
        - name: MONITOR_INTERVAL_SECONDS
          value: "30"
        - name: SCALE_UP_DELAY_SECONDS
          value: "60"
        - name: SCALE_DOWN_DELAY_SECONDS
          value: "120"
        - name: MAX_MEMORY_MB
          value: "2048"
        - name: NETWORK_INTERFACE
          value: "eth0"
        resources:
          requests:
            memory: "100Mi"
            cpu: "50m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
        securityContext:
          privileged: false
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 1000
          capabilities:
            drop:
            - ALL
      tolerations:
      - operator: Exists
        effect: NoSchedule
      - operator: Exists
        effect: NoExecute
      nodeSelector:
        kubernetes.io/os: linux
        kubernetes.io/arch: arm64
      priorityClassName: system-node-critical
EOF
}

show_status() {
    echo "📊 goburn Deployment Status"
    echo "============================"
    
    echo ""
    echo "🏗️  RBAC Resources:"
    kubectl get serviceaccount goburn -n $NAMESPACE 2>/dev/null && echo "✅ ServiceAccount exists" || echo "❌ ServiceAccount missing"
    kubectl get clusterrole goburn 2>/dev/null && echo "✅ ClusterRole exists" || echo "❌ ClusterRole missing"
    kubectl get clusterrolebinding goburn 2>/dev/null && echo "✅ ClusterRoleBinding exists" || echo "❌ ClusterRoleBinding missing"
    
    echo ""
    echo "🖥️  AMD64 Nodes (CPU + Network only):"
    AMD64_DESIRED=$(kubectl get daemonset goburn-amd64 -n $NAMESPACE -o jsonpath='{.status.desiredNumberScheduled}' 2>/dev/null || echo "0")
    AMD64_READY=$(kubectl get daemonset goburn-amd64 -n $NAMESPACE -o jsonpath='{.status.numberReady}' 2>/dev/null || echo "0")
    echo "   Desired: $AMD64_DESIRED, Ready: $AMD64_READY"
    
    if [ "$AMD64_READY" -gt 0 ]; then
        echo "   Nodes:"
        kubectl get pods -l app=goburn,arch=amd64 -n $NAMESPACE -o wide --no-headers | awk '{print "   - " $7 " (" $3 ")"}'
    fi
    
    echo ""
    echo "💪 ARM64 Nodes (CPU + Network + Memory):"
    ARM64_DESIRED=$(kubectl get daemonset goburn-arm64 -n $NAMESPACE -o jsonpath='{.status.desiredNumberScheduled}' 2>/dev/null || echo "0")
    ARM64_READY=$(kubectl get daemonset goburn-arm64 -n $NAMESPACE -o jsonpath='{.status.numberReady}' 2>/dev/null || echo "0")
    echo "   Desired: $ARM64_DESIRED, Ready: $ARM64_READY"
    
    if [ "$ARM64_READY" -gt 0 ]; then
        echo "   Nodes:"
        kubectl get pods -l app=goburn,arch=arm64 -n $NAMESPACE -o wide --no-headers | awk '{print "   - " $7 " (" $3 ")"}'
    fi
    
    echo ""
    echo "📋 Recent Events:"
    kubectl get events -n $NAMESPACE --field-selector involvedObject.kind=DaemonSet --sort-by='.lastTimestamp' | tail -5
    
    echo ""
    echo "💡 Useful Commands:"
    echo "   kubectl logs -l app=goburn,arch=amd64 -n $NAMESPACE -f"
    echo "   kubectl logs -l app=goburn,arch=arm64 -n $NAMESPACE -f"
    echo "   kubectl get pods -l app=goburn -n $NAMESPACE -o wide"
}

cleanup() {
    echo "🧹 Cleaning up goburn resources..."
    kubectl delete daemonset goburn-amd64 -n $NAMESPACE 2>/dev/null || echo "AMD64 DaemonSet not found"
    kubectl delete daemonset goburn-arm64 -n $NAMESPACE 2>/dev/null || echo "ARM64 DaemonSet not found"
    kubectl delete clusterrolebinding goburn 2>/dev/null || echo "ClusterRoleBinding not found"
    kubectl delete clusterrole goburn 2>/dev/null || echo "ClusterRole not found"
    kubectl delete serviceaccount goburn -n $NAMESPACE 2>/dev/null || echo "ServiceAccount not found"
    echo "✅ Cleanup complete"
}

case "${1:-both}" in
    "amd64")
        deploy_rbac
        deploy_amd64
        echo "✅ AMD64 deployment complete"
        ;;
    "arm64")
        deploy_rbac
        deploy_arm64
        echo "✅ ARM64 deployment complete"
        ;;
    "both")
        deploy_rbac
        deploy_amd64
        deploy_arm64
        echo "✅ Both architectures deployed"
        ;;
    "status")
        show_status
        ;;
    "cleanup")
        cleanup
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    *)
        echo "❌ Unknown command: $1"
        show_help
        exit 1
        ;;
esac
