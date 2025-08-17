# Kaiwo-PoC Development Environment Setup
## Local Development + Remote GPU Validation

---

## Overview

This guide sets up a development environment where you:
- **Local development**: Code, build, and test locally
- **Validate on remote node**: Deploy and test with real Instinct GPUs

---

## Step 1: Local Development Environment Setup

### 1.1 Install Required Tools

```bash
# Navigate to your project
cd ~/Desktop/kaiwo-PoC

# Install Go (if not already installed)
# Download from https://golang.org/dl/
# or use package manager:
sudo apt update
sudo apt install golang-go

# Install Docker Desktop
# Download from https://www.docker.com/products/docker-desktop

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# Install additional development tools
sudo apt install make git curl wget
```

### 1.2 Configure Go Environment

```bash
# Set up Go workspace
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# Add to your ~/.bashrc or ~/.zshrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
source ~/.bashrc
```

### 1.3 Install Development Tools

```bash
# Install Go tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
go install k8s.io/code-generator/cmd/...@latest

# Install VS Code extensions (if using VS Code)
code --install-extension golang.go
code --install-extension ms-kubernetes-tools.vscode-kubernetes-tools
code --install-extension redhat.vscode-yaml
```

---

## Step 2: Remote Node Setup

### 2.1 Prerequisites for Remote Node

Your remote node should have:
- **Kubernetes cluster** (can be single-node for development)
- **AMD Instinct GPUs** with proper drivers
- **AMD GPU Operator** installed
- **Network connectivity** from your laptop

### 2.2 Install AMD GPU Operator on Remote Node

```bash
# On the remote node, install AMD GPU Operator
helm repo add amd-gpu-operator https://rocm.github.io/amd-gpu-operator
helm repo update

# Install the operator
helm install amd-gpu-operator amd-gpu-operator/amd-gpu-operator \
  --namespace gpu-operator-resources \
  --create-namespace \
  --set devicePlugin.enabled=true \
  --set nodeFeatureDiscovery.enabled=true
```

### 2.3 Verify GPU Detection

```bash
# Check if GPUs are detected
kubectl get nodes -o json | jq '.items[].status.allocatable | keys | .[] | select(contains("amd.com/gpu"))'

# Should show something like:
# "amd.com/gpu": "4"
```

---

## Step 3: Connect Laptop to Remote Cluster

### 3.1 Configure kubectl for Remote Cluster

```bash
# Copy kubeconfig from remote node to your laptop
# Option 1: If you have SSH access
scp user@remote-node:/etc/kubernetes/admin.conf ~/.kube/config

# Option 2: If using cloud provider
# Download kubeconfig from your cloud provider's dashboard

# Option 3: If using kind/minikube on remote
scp user@remote-node:~/.kube/config ~/.kube/config

# Test connection
kubectl get nodes
kubectl get pods --all-namespaces
```

### 3.2 Set up SSH Key-based Authentication (Optional but Recommended)

```bash
# Generate SSH key if you don't have one
ssh-keygen -t rsa -b 4096 -C "your-email@example.com"

# Copy to remote node
ssh-copy-id user@remote-node

# Test SSH connection
ssh user@remote-node "echo 'SSH connection successful'"
```

---

## Step 4: Development Workflow Setup

### 4.1 Local Development Commands

```bash
# Navigate to your project
cd ~/Desktop/kaiwo-PoC

# Build locally
make build

# Run unit tests
make test

# Generate code
make generate

# Build Docker image locally
make docker-build

# Run linting
make lint
```

### 4.2 Remote Deployment Scripts

Create deployment scripts for easy remote testing:

```bash
# Create ~/Desktop/kaiwo-PoC/scripts/deploy-remote.sh
#!/bin/bash
set -e

echo "Building Kaiwo-PoC..."
make build

echo "Building Docker image..."
make docker-build

echo "Tagging image for remote registry..."
docker tag kaiwo-poc:latest your-registry.com/kaiwo-poc:latest

echo "Pushing to remote registry..."
docker push your-registry.com/kaiwo-poc:latest

echo "Deploying to remote cluster..."
kubectl apply -f config/default/

echo "Waiting for deployment..."
kubectl wait --for=condition=available --timeout=300s deployment/kaiwo-poc-controller-manager -n kaiwo-poc-system

echo "Deployment complete!"
```

Make it executable:
```bash
chmod +x ~/Desktop/kaiwo-PoC/scripts/deploy-remote.sh
```

### 4.3 Quick Validation Script

```bash
# Create ~/Desktop/kaiwo-PoC/scripts/validate-gpu.sh
#!/bin/bash
set -e

echo "Testing GPU detection..."
kubectl get nodes -o json | jq '.items[].status.allocatable | keys | .[] | select(contains("amd.com/gpu"))'

echo "Deploying test workload..."
kubectl apply -f test/manifests/test-gpu-job.yaml

echo "Monitoring test workload..."
kubectl logs -f job/test-gpu-job

echo "Cleaning up..."
kubectl delete -f test/manifests/test-gpu-job.yaml
```

---

## Step 5: Development Workflow

### 5.1 Daily Development Cycle

```bash
# 1. Make changes locally
cd ~/Desktop/kaiwo-PoC
# Edit code in your preferred editor

# 2. Test locally
make test
make lint

# 3. Build and deploy to remote
./scripts/deploy-remote.sh

# 4. Validate on remote GPU node
./scripts/validate-gpu.sh

# 5. Iterate based on results
```

### 5.2 Debugging Workflow

```bash
# View logs from remote cluster
kubectl logs -f deployment/kaiwo-poc-controller-manager -n kaiwo-poc-system

# Check GPU operator logs
kubectl logs -f deployment/amd-gpu-operator-device-plugin -n gpu-operator-resources

# Check node GPU status
kubectl describe node <gpu-node-name>

# SSH to remote node for direct debugging
ssh user@remote-node
# Then check GPU status, logs, etc.
```

---

## Step 6: Testing with Real GPU Workloads

### 6.1 Create Test GPU Job

```yaml
# Create ~/Desktop/kaiwo-PoC/test/manifests/test-gpu-job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: test-gpu-job
spec:
  template:
    spec:
      containers:
      - name: gpu-test
        image: rocm/pytorch:latest
        command: ["python", "-c", "
import torch
print(f'CUDA available: {torch.cuda.is_available()}')
print(f'GPU count: {torch.cuda.device_count()}')
if torch.cuda.is_available():
    print(f'GPU name: {torch.cuda.get_device_name(0)}')
    print(f'GPU memory: {torch.cuda.get_device_properties(0).total_memory / 1024**3:.1f} GB')
"]
        resources:
          limits:
            amd.com/gpu: 1
      restartPolicy: Never
  backoffLimit: 4
```

### 6.2 Create Kaiwo-PoC Test Job

```yaml
# Create ~/Desktop/kaiwo-PoC/test/manifests/test-kaiwo-job.yaml
apiVersion: kaiwo.ai/v1alpha1
kind: KaiwoJob
metadata:
  name: test-kaiwo-gpu-job
spec:
  image: rocm/pytorch:latest
  gpus: 1
  replicas: 1
  entrypoint: |
    python -c "
import torch
print(f'CUDA available: {torch.cuda.is_available()}')
print(f'GPU count: {torch.cuda.device_count()}')
if torch.cuda.is_available():
    print(f'GPU name: {torch.cuda.get_device_name(0)}')
    print(f'GPU memory: {torch.cuda.get_device_properties(0).total_memory / 1024**3:.1f} GB')
"
```

---

## Step 7: Performance Monitoring

### 7.1 Set up Monitoring Tools

```bash
# Install Prometheus and Grafana on remote cluster
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack

# Access Grafana
kubectl port-forward svc/prometheus-grafana 3000:80 -n default
# Then visit http://localhost:3000 (admin/prom-operator)
```

### 7.2 GPU Metrics Dashboard

Create a Grafana dashboard for GPU monitoring:

```json
{
  "dashboard": {
    "title": "AMD GPU Metrics",
    "panels": [
      {
        "title": "GPU Utilization",
        "type": "graph",
        "targets": [
          {
            "expr": "amd_gpu_utilization",
            "legendFormat": "GPU {{gpu_id}}"
          }
        ]
      },
      {
        "title": "GPU Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "amd_gpu_memory_used_bytes",
            "legendFormat": "GPU {{gpu_id}}"
          }
        ]
      }
    ]
  }
}
```

---

## Step 8: Troubleshooting

### 8.1 Common Issues and Solutions

#### GPU Not Detected
```bash
# Check GPU operator status
kubectl get pods -n gpu-operator-resources

# Check node labels
kubectl get nodes --show-labels | grep gpu

# Check GPU device plugin
kubectl describe pod -n gpu-operator-resources -l app=amd-gpu-operator-device-plugin
```

#### Connection Issues
```bash
# Test kubectl connection
kubectl cluster-info

# Check kubeconfig
kubectl config view

# Test SSH connection
ssh user@remote-node "kubectl get nodes"
```

#### Build Issues
```bash
# Clean and rebuild
make clean
make build

# Check Go version
go version

# Update dependencies
go mod tidy
go mod download
```

---

## Step 9: Advanced Development Features

### 9.1 Hot Reload Development

```bash
# Install skaffold for hot reload
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64
chmod +x skaffold
sudo mv skaffold /usr/local/bin

# Create skaffold.yaml for hot reload
cat > ~/Desktop/kaiwo-PoC/skaffold.yaml << EOF
apiVersion: skaffold/v2beta29
kind: Config
build:
  artifacts:
  - image: kaiwo-poc
    docker:
      dockerfile: Dockerfile
deploy:
  kubectl:
    manifests:
    - config/default/
EOF

# Run with hot reload
skaffold dev
```

### 9.2 Remote Debugging

```bash
# Enable remote debugging in your IDE
# Add to your deployment:
# - name: dlv
#   image: golang:1.21
#   command: ["dlv", "debug", "--headless", "--listen=:2345", "--api-version=2", "--accept-multiclient"]
#   ports:
#   - containerPort: 2345

# Port forward for debugging
kubectl port-forward deployment/kaiwo-poc-controller-manager 2345:2345 -n kaiwo-poc-system
```

---
