#!/bin/bash

set -e

# Build script for goburn - Dynamic Kubernetes Resource Utilization Tool

echo "🔥 Building goburn..."

# Generate go.sum if it doesn't exist
if [ ! -f "go.sum" ]; then
    echo "📦 Downloading dependencies..."
    go mod tidy
fi

# Build the Docker image
echo "🐳 Building Docker image..."
docker build -t pedromol/goburn:latest .

# Tag with version if provided
if [ "$1" != "" ]; then
    echo "🏷️  Tagging with version $1..."
    docker tag pedromol/goburn:latest pedromol/goburn:$1
fi

echo "✅ Build complete!"
echo ""
echo "To deploy to Kubernetes:"
echo "  kubectl apply -f k8s-manifests.yaml"
echo ""
echo "To test locally:"
echo "  docker-compose up -d"
echo ""
echo "To push to registry:"
echo "  docker push pedromol/goburn:latest"
if [ "$1" != "" ]; then
    echo "  docker push pedromol/goburn:$1"
fi
