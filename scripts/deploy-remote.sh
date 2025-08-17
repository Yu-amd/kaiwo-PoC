#!/bin/bash
set -e

echo "🚀 Deploying Kaiwo-PoC to remote cluster..."

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "❌ Error: Please run this script from the kaiwo-PoC root directory"
    exit 1
fi

echo "📦 Building Kaiwo-PoC..."
make build

echo "🐳 Building Docker image..."
make docker-build

echo "🏷️  Tagging image for remote registry..."
# Update this with your actual registry
REGISTRY=${REGISTRY:-"your-registry.com"}
IMAGE_TAG=${IMAGE_TAG:-"latest"}
docker tag kaiwo-poc:latest ${REGISTRY}/kaiwo-poc:${IMAGE_TAG}

echo "⬆️  Pushing to remote registry..."
docker push ${REGISTRY}/kaiwo-poc:${IMAGE_TAG}

echo "🚀 Deploying to remote cluster..."
kubectl apply -f config/default/

echo "⏳ Waiting for deployment..."
kubectl wait --for=condition=available --timeout=300s deployment/kaiwo-poc-controller-manager -n kaiwo-poc-system

echo "✅ Deployment complete!"
echo "🔍 Check deployment status:"
kubectl get pods -n kaiwo-poc-system
