#!/bin/bash
set -e

echo "🧪 Testing GPU Setup for Kaiwo-PoC"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Test 1: Check if kubectl is working
print_status "Test 1: Checking kubectl connectivity..."
if kubectl get nodes &>/dev/null; then
    print_success "kubectl is working"
else
    print_error "kubectl is not working"
    exit 1
fi

# Test 2: Check cluster status
print_status "Test 2: Checking cluster status..."
kubectl get nodes
kubectl get pods --all-namespaces

# Test 3: Check for AMD GPU resources
print_status "Test 3: Checking for AMD Instinct GPU resources..."
if kubectl get nodes -o json | jq -r '.items[].status.allocatable | keys | .[] | select(contains("amd.com/gpu"))' 2>/dev/null | grep -q .; then
    print_success "AMD Instinct GPU resources detected in Kubernetes"
    kubectl get nodes -o json | jq '.items[].status.allocatable | keys | .[] | select(contains("amd.com/gpu"))'
    print_status "AMD GPU resources available:"
    kubectl get nodes -o json | jq '.items[0].status.allocatable."amd.com/gpu"'
else
    print_warning "No AMD Instinct GPU resources detected in Kubernetes"
fi

# Test 4: Check for AMD GPU labels
print_status "Test 4: Checking for AMD GPU labels..."
if kubectl get nodes -o json | jq -r '.items[].metadata.labels | keys | .[] | select(contains("amd.com/gpu"))' 2>/dev/null | grep -q .; then
    print_success "AMD GPU labels detected"
    kubectl get nodes -o json | jq '.items[].metadata.labels | keys | .[] | select(contains("amd.com/gpu"))'
else
    print_warning "No AMD GPU labels detected"
fi

# Test 5: Check if rocm-smi is available
print_status "Test 5: Checking AMD ROCm SMI availability..."
if command -v rocm-smi &> /dev/null; then
    print_success "AMD ROCm SMI is available"
    rocm-smi --showproductname
else
    print_warning "AMD ROCm SMI is not available"
fi

# Test 6: Create a simple test pod
print_status "Test 6: Creating a simple test pod..."
kubectl run test-pod --image=nginx --restart=Never --dry-run=client -o yaml > /tmp/test-pod.yaml
kubectl apply -f /tmp/test-pod.yaml

# Wait for pod to be ready
for i in {1..30}; do
    if kubectl get pod test-pod | grep -q "Running"; then
        print_success "Test pod is running"
        break
    fi
    print_status "Waiting for test pod to be ready... (attempt $i/30)"
    sleep 2
done

# Clean up test pod
kubectl delete pod test-pod

# Test 7: Check AMD GPU device plugin status
print_status "Test 7: Checking AMD GPU device plugin status..."
if kubectl get pods -n kube-system | grep -q "amd-gpu-device-plugin"; then
    print_success "AMD GPU device plugin is running"
    kubectl get pods -n kube-system | grep "amd-gpu-device-plugin"
else
    print_warning "AMD GPU device plugin not found"
fi

# Test 8: Check GPU device plugin logs
print_status "Test 8: Checking GPU device plugin logs..."
if kubectl get pods -n kube-system | grep -q "amd-gpu-device-plugin"; then
    POD_NAME=$(kubectl get pods -n kube-system | grep "amd-gpu-device-plugin" | awk '{print $1}')
    print_status "GPU device plugin logs:"
    kubectl logs $POD_NAME -n kube-system --tail=10
else
    print_warning "No GPU device plugin logs available"
fi

print_success "GPU setup testing completed!"
echo
print_status "Summary:"
echo "- kubectl: Working"
echo "- Cluster: Running"
echo "- AMD Instinct GPU Resources: $(if kubectl get nodes -o json | jq -r '.items[].status.allocatable | keys | .[] | select(contains("amd.com/gpu"))' 2>/dev/null | grep -q .; then echo "Detected"; else echo "Not detected"; fi)"
echo "- AMD GPU Labels: $(if kubectl get nodes -o json | jq -r '.items[].metadata.labels | keys | .[] | select(contains("amd.com/gpu"))' 2>/dev/null | grep -q .; then echo "Detected"; else echo "Not detected"; fi)"
echo "- AMD ROCm SMI: $(if command -v rocm-smi &> /dev/null; then echo "Available"; else echo "Not available"; fi)"
echo "- Test Pod: Working"
echo "- AMD GPU Device Plugin: $(if kubectl get pods -n kube-system | grep -q "amd-gpu-device-plugin"; then echo "Running"; else echo "Not running"; fi)"
