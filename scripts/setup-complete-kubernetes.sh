#!/bin/bash
set -e

echo "🚀 Setting up Complete Kubernetes Cluster for Kaiwo-PoC Development Environment"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print colored output
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

print_step() {
    echo -e "${PURPLE}[STEP]${NC} $1"
}

print_header() {
    echo -e "${CYAN}================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}================================${NC}"
}

# Check if running as root
#if [[ $EUID -eq 0 ]]; then
#   print_error "This script should not be run as root. Please run as a regular user with sudo privileges."
#   exit 1
#fi

# Check if we're on the right system
if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    print_error "This script is designed for Linux systems only."
    exit 1
fi

print_header "Kaiwo-PoC Kubernetes Setup"
print_status "This script will set up a complete Kubernetes cluster with AMD GPU support"
print_status "for your Kaiwo-PoC development environment."

# Detect Linux distribution
print_step "Detecting Linux distribution..."
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$NAME
    VER=$VERSION_ID
    print_status "Detected: $OS $VER"
else
    print_error "Cannot detect Linux distribution"
    exit 1
fi

# Check system requirements
print_step "Checking system requirements..."

# Check available memory (need at least 2GB)
MEMORY_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}')
MEMORY_GB=$((MEMORY_KB / 1024 / 1024))
if [ "$MEMORY_GB" -lt 2 ]; then
    print_error "Insufficient memory. Need at least 2GB, found ${MEMORY_GB}GB"
    exit 1
fi
print_success "Memory: ${MEMORY_GB}GB (✓)"

# Check available disk space (need at least 10GB)
DISK_SPACE_GB=$(df / | tail -1 | awk '{print $4 / 1024 / 1024}' | cut -d. -f1)
if [ "$DISK_SPACE_GB" -lt 10 ]; then
    print_error "Insufficient disk space. Need at least 10GB, found ${DISK_SPACE_GB}GB"
    exit 1
fi
print_success "Disk space: ${DISK_SPACE_GB}GB (✓)"

# Check if running in a VM or bare metal
if systemd-detect-virt -q; then
    VIRT_TYPE=$(systemd-detect-virt)
    print_warning "Running in virtual environment: $VIRT_TYPE"
    print_warning "GPU support may be limited in virtual environments"
else
    print_success "Running on bare metal (✓)"
fi

# Check for AMD GPUs
print_step "Checking for AMD GPUs..."
if command -v lspci &> /dev/null; then
    GPU_COUNT=$(lspci | grep -i "VGA\|3D\|Display" | grep -i amd | wc -l)
    if [ "$GPU_COUNT" -gt 0 ]; then
        print_success "Found $GPU_COUNT AMD GPU(s)"
        lspci | grep -i "VGA\|3D\|Display" | grep -i amd
    else
        print_warning "No AMD GPUs detected. GPU workloads will not be available."
    fi
else
    print_warning "lspci not available, cannot detect GPUs"
fi

# Ask for confirmation before proceeding
echo
print_warning "This will install a complete Kubernetes cluster with AMD GPU support."
print_warning "This process may take 10-20 minutes depending on your system and internet connection."
echo
read -p "Do you want to proceed? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_status "Setup cancelled by user"
    exit 0
fi

# Backup existing kubeconfig if it exists
if [ -f ~/.kube/config ]; then
    print_status "Backing up existing kubeconfig..."
    cp ~/.kube/config ~/.kube/config.backup.$(date +%Y%m%d_%H%M%S)
    print_success "Backup created"
fi

# Update system packages
print_step "Updating system packages..."
sudo apt update
print_success "System packages updated"

# Install required packages
print_step "Installing required packages..."
sudo apt install -y curl wget git jq apt-transport-https ca-certificates gnupg lsb-release
print_success "Required packages installed"

# Install Docker
print_step "Installing Docker..."
if ! command -v docker &> /dev/null; then
    # Add Docker's official GPG key
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
    
    # Add Docker repository
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
    
    # Install Docker
    sudo apt update
    sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    
    # Add user to docker group
    sudo usermod -aG docker $USER
    print_success "Docker installed"
else
    print_success "Docker already installed"
fi

# Start and enable Docker
sudo systemctl start docker
sudo systemctl enable docker

# Install kubectl
print_step "Installing kubectl..."
if ! command -v kubectl &> /dev/null; then
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    chmod +x kubectl
    sudo mv kubectl /usr/local/bin/
    print_success "kubectl installed"
else
    print_success "kubectl already installed"
fi

# Install Helm
print_step "Installing Helm..."
if ! command -v helm &> /dev/null; then
    curl https://baltocdn.com/helm/signing.asc | gpg --dearmor | sudo tee /usr/share/keyrings/helm.gpg > /dev/null
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/helm.gpg] https://baltocdn.com/helm/stable/debian/ all main" | sudo tee /etc/apt/sources.list.d/helm-stable-debian.list
    sudo apt update
    sudo apt install -y helm
    print_success "Helm installed"
else
    print_success "Helm already installed"
fi

# Install vanilla Kubernetes using kubeadm
print_step "Installing vanilla Kubernetes using kubeadm..."

# Disable swap (required for kubelet)
print_status "Disabling swap..."
sudo swapoff -a
sudo sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab

# Install Kubernetes components
if ! command -v kubeadm &> /dev/null; then
    # Add Kubernetes repository
    curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.29/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
    echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.29/deb/ /' | sudo tee /etc/apt/sources.list.d/kubernetes.list
    
    # Install Kubernetes components
    sudo apt update
    sudo apt install -y kubelet kubeadm kubectl
    sudo apt-mark hold kubelet kubeadm kubectl
    
    print_success "Kubernetes components installed"
else
    print_success "Kubernetes components already installed"
fi

# Configure containerd for Kubernetes
print_status "Configuring containerd..."
sudo mkdir -p /etc/containerd
containerd config default | sudo tee /etc/containerd/config.toml
sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml
sudo systemctl restart containerd

# Initialize Kubernetes cluster
print_step "Initializing Kubernetes cluster..."
if [ ! -f /etc/kubernetes/admin.conf ]; then
    # Get the primary IP address
    PRIMARY_IP=$(ip route get 1 | awk '{print $7; exit}')
    
    # Initialize the cluster
    sudo kubeadm init --pod-network-cidr=10.244.0.0/16 --apiserver-advertise-address=$PRIMARY_IP
    
    # Set up kubectl for the current user
    mkdir -p ~/.kube
    sudo cp /etc/kubernetes/admin.conf ~/.kube/config
    sudo chown $USER:$USER ~/.kube/config
    
    print_success "Kubernetes cluster initialized"
else
    print_success "Kubernetes cluster already initialized"
    
    # Set up kubectl for the current user if not already done
    if [ ! -f ~/.kube/config ]; then
        mkdir -p ~/.kube
        sudo cp /etc/kubernetes/admin.conf ~/.kube/config
        sudo chown $USER:$USER ~/.kube/config
    fi
fi

# Wait for Kubernetes to be ready
print_status "Waiting for Kubernetes to be ready..."
sleep 60

# Verify Kubernetes installation
print_step "Verifying Kubernetes installation..."
if kubectl get nodes; then
    print_success "Kubernetes cluster is running"
else
    print_error "Kubernetes cluster is not running properly"
    exit 1
fi

# Install CNI (Flannel)
print_step "Installing CNI (Flannel)..."
kubectl apply -f https://raw.githubusercontent.com/flannel-io/flannel/master/Documentation/kube-flannel.yml
print_success "Flannel CNI installed"

# Wait for CNI to be ready
print_status "Waiting for CNI to be ready..."
sleep 60

# Remove taint from master node to allow scheduling
print_status "Configuring master node for single-node cluster..."
kubectl taint nodes --all node-role.kubernetes.io/control-plane-
kubectl taint nodes --all node-role.kubernetes.io/master-

# Install AMD GPU drivers (if AMD GPUs are present)
if [ "$GPU_COUNT" -gt 0 ]; then
    print_step "Installing AMD GPU drivers..."
    
    # Add ROCm repository
    sudo mkdir -p /etc/apt/keyrings
    curl -fsSL https://repo.radeon.com/rocm/rocm.gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/rocm.gpg
    
    echo 'deb [arch=amd64 signed-by=/etc/apt/keyrings/rocm.gpg] https://repo.radeon.com/rocm/apt/debian jammy main' | sudo tee /etc/apt/sources.list.d/rocm.list
    echo -e 'Package: *\nPin: release o=repo.radeon.com\nPin-Priority: 600' | sudo tee /etc/apt/preferences.d/rocm-pin-600
    
    sudo apt update
    sudo apt install -y rocm-dkms
    
    print_success "AMD GPU drivers installed"
    
    # Verify GPU detection
    if command -v rocm-smi &> /dev/null; then
        print_status "Verifying GPU detection..."
        rocm-smi
    fi
fi

# Install AMD GPU Operator
print_step "Installing AMD GPU Operator..."
helm repo add amd-gpu-operator https://rocm.github.io/amd-gpu-operator
helm repo update

helm install amd-gpu-operator amd-gpu-operator/amd-gpu-operator \
  --namespace gpu-operator-resources \
  --create-namespace \
  --wait

print_success "AMD GPU Operator installed"

# Wait for GPU operator to be ready
print_status "Waiting for AMD GPU Operator to be ready..."
sleep 60

# Install additional tools for development
print_step "Installing additional development tools..."

# Install k9s (Kubernetes CLI tool)
if ! command -v k9s &> /dev/null; then
    curl -sS https://webinstall.dev/k9s | bash
    print_success "k9s installed"
else
    print_success "k9s already installed"
fi

# Install kubectx and kubens
if ! command -v kubectx &> /dev/null; then
    sudo git clone https://github.com/ahmetb/kubectx /opt/kubectx
    sudo ln -s /opt/kubectx/kubectx /usr/local/bin/kubectx
    sudo ln -s /opt/kubectx/kubens /usr/local/bin/kubens
    print_success "kubectx and kubens installed"
else
    print_success "kubectx and kubens already installed"
fi

# Install kubectl plugins
print_step "Installing kubectl plugins..."
mkdir -p ~/.kube/plugins

# Install kubectl tree plugin
if ! kubectl tree --help &> /dev/null; then
    curl -L https://github.com/ahmetb/kubectl-tree/releases/download/v0.2.1/kubectl-tree_v0.2.1_linux_amd64.tar.gz | tar -xz
    sudo mv kubectl-tree /usr/local/bin/
    print_success "kubectl tree plugin installed"
else
    print_success "kubectl tree plugin already installed"
fi

# Verify the installation
print_step "Verifying complete installation..."

# Check cluster status
if kubectl get nodes; then
    print_success "Kubernetes cluster is running"
else
    print_error "Failed to get cluster nodes"
    exit 1
fi

# Check system pods
if kubectl get pods --all-namespaces; then
    print_success "System pods are running"
else
    print_error "Failed to get system pods"
    exit 1
fi

# Check for GPU support
print_step "Checking GPU support..."
if kubectl get nodes -o json | jq -r '.items[].status.allocatable | keys | .[] | select(contains("amd.com/gpu"))' 2>/dev/null | grep -q .; then
    print_success "AMD GPU support detected!"
    kubectl get nodes -o json | jq '.items[].status.allocatable | keys | .[] | select(contains("amd.com/gpu"))'
else
    print_warning "No AMD GPU support detected. This is normal if no AMD GPUs are present."
fi

# Create a comprehensive summary file
SUMMARY_FILE="$HOME/kaiwo-poc-k8s-setup-summary.txt"
cat > "$SUMMARY_FILE" << EOF
Kaiwo-PoC Kubernetes Development Environment Setup Summary
==========================================================
Date: $(date)
Setup Script: kaiwo-PoC/scripts/setup-complete-kubernetes.sh

System Information:
- OS: $OS $VER
- Memory: ${MEMORY_GB}GB
- Disk Space: ${DISK_SPACE_GB}GB
- AMD GPUs: $GPU_COUNT

Cluster Information:
$(kubectl cluster-info 2>/dev/null || echo "Cluster info not available")

Nodes:
$(kubectl get nodes 2>/dev/null || echo "Nodes not available")

GPU Resources:
$(kubectl get nodes -o json | jq '.items[].status.allocatable | keys | .[] | select(contains("amd.com/gpu"))' 2>/dev/null || echo "No GPU resources detected")

Installed Components:
- Vanilla Kubernetes (kubeadm)
- Docker
- kubectl
- Helm
- AMD GPU Operator
- Flannel CNI
- k9s (Kubernetes CLI)
- kubectx/kubens
- kubectl tree plugin

Next Steps for Kaiwo-PoC Development:
1. Test GPU functionality: kubectl apply -f test/manifests/test-gpu-job.yaml
2. Deploy Kaiwo-PoC: ./scripts/deploy-remote.sh
3. Validate setup: ./scripts/validate-gpu.sh
4. Start development: make build && make test

Useful Commands:
- View cluster: kubectl get nodes
- View pods: kubectl get pods --all-namespaces
- Interactive cluster view: k9s
- Switch contexts: kubectx
- Switch namespaces: kubens
- View resource tree: kubectl tree

kubeconfig location: ~/.kube/config
EOF

print_success "Setup summary saved to: $SUMMARY_FILE"

# Create a quick reference card
REFERENCE_FILE="$HOME/kaiwo-poc-quick-reference.txt"
cat > "$REFERENCE_FILE" << EOF
Kaiwo-PoC Development Quick Reference
====================================

Cluster Management:
  kubectl get nodes                    # View cluster nodes
  kubectl get pods --all-namespaces    # View all pods
  k9s                                  # Interactive cluster view
  kubectx                              # Switch contexts
  kubens                               # Switch namespaces

GPU Management:
  kubectl get nodes -o json | jq '.items[].status.allocatable | keys | .[] | select(contains("amd.com/gpu"))'
  rocm-smi                             # AMD GPU status
  kubectl get pods -n gpu-operator-resources

Kaiwo-PoC Development:
  make build                           # Build Kaiwo-PoC
  make test                            # Run tests
  ./scripts/deploy-remote.sh           # Deploy to cluster
  ./scripts/validate-gpu.sh            # Test GPU functionality

Troubleshooting:
  kubectl logs -f <pod-name>           # View pod logs
  kubectl describe pod <pod-name>      # Describe pod
  kubectl exec -it <pod-name> -- bash  # Enter pod
  kubectl get events --all-namespaces  # View events
EOF

print_success "Quick reference saved to: $REFERENCE_FILE"

# Display completion message
print_header "🎉 Setup Complete!"
print_success "Your Kaiwo-PoC Kubernetes development environment is ready!"

echo
print_status "What's been installed:"
echo "  ✅ Vanilla Kubernetes (production-ready cluster)"
echo "  ✅ Docker (container runtime)"
echo "  ✅ kubectl (Kubernetes CLI)"
echo "  ✅ Helm (package manager)"
echo "  ✅ AMD GPU Operator (GPU management)"
echo "  ✅ Flannel CNI (networking)"
echo "  ✅ k9s (interactive cluster view)"
echo "  ✅ kubectx/kubens (context/namespace switching)"
echo "  ✅ kubectl tree plugin (resource visualization)"

echo
print_status "Next steps:"
echo "1. Test GPU functionality:"
echo "   kubectl apply -f test/manifests/test-gpu-job.yaml"
echo
echo "2. Deploy Kaiwo-PoC:"
echo "   ./scripts/deploy-remote.sh"
echo
echo "3. Validate your setup:"
echo "   ./scripts/validate-gpu.sh"
echo
echo "4. Start developing:"
echo "   make build && make test"
echo
echo "5. View cluster interactively:"
echo "   k9s"
echo
echo "6. Check setup summary:"
echo "   cat $SUMMARY_FILE"
echo
echo "7. Quick reference:"
echo "   cat $REFERENCE_FILE"

echo
print_status "Your development environment is ready! 🚀"
print_status "Happy coding with Kaiwo-PoC! 🎯"
