#!/bin/bash
set -e

echo "🚀 Setting up Local Development Environment for Kaiwo-PoC"

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
if [[ $EUID -eq 0 ]]; then
   print_error "This script should not be run as root. Please run as a regular user with sudo privileges."
   exit 1
fi

# Check if we're on the right system
if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    print_error "This script is designed for Linux systems only."
    exit 1
fi

print_header "Kaiwo-PoC Local Development Setup"
print_status "This script will set up your local development environment for Kaiwo-PoC development."

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

# Check available memory (need at least 4GB for comfortable development)
MEMORY_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}')
MEMORY_GB=$((MEMORY_KB / 1024 / 1024))
if [ "$MEMORY_GB" -lt 4 ]; then
    print_warning "Low memory detected: ${MEMORY_GB}GB. Recommended: 8GB+ for comfortable development."
else
    print_success "Memory: ${MEMORY_GB}GB (✓)"
fi

# Check available disk space (need at least 5GB)
DISK_SPACE_GB=$(df / | tail -1 | awk '{print $4 / 1024 / 1024}' | cut -d. -f1)
if [ "$DISK_SPACE_GB" -lt 5 ]; then
    print_error "Insufficient disk space. Need at least 5GB, found ${DISK_SPACE_GB}GB"
    exit 1
fi
print_success "Disk space: ${DISK_SPACE_GB}GB (✓)"

# Ask for confirmation before proceeding
echo
print_warning "This will install development tools for Kaiwo-PoC development."
print_warning "This process may take 5-10 minutes depending on your system and internet connection."
echo
read -p "Do you want to proceed? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_status "Setup cancelled by user"
    exit 0
fi

# Update system packages
print_step "Updating system packages..."
sudo apt update
print_success "System packages updated"

# Install essential packages
print_step "Installing essential packages..."
sudo apt install -y curl wget git jq apt-transport-https ca-certificates gnupg lsb-release build-essential
print_success "Essential packages installed"

# Install Go
print_step "Installing Go..."
if ! command -v go &> /dev/null; then
    # Download latest Go
    GO_VERSION=$(curl -s https://golang.org/dl/ | grep -o 'go[0-9]\+\.[0-9]\+\.[0-9]\+' | head -1)
    GO_ARCH="linux-amd64"
    GO_TAR="${GO_VERSION}.${GO_ARCH}.tar.gz"
    
    print_status "Downloading Go ${GO_VERSION}..."
    curl -LO "https://golang.org/dl/${GO_TAR}"
    
    # Remove old Go installation if exists
    sudo rm -rf /usr/local/go
    
    # Install Go
    sudo tar -C /usr/local -xzf "${GO_TAR}"
    rm "${GO_TAR}"
    
    # Add Go to PATH
    if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        export PATH=$PATH:/usr/local/go/bin
    fi
    
    print_success "Go ${GO_VERSION} installed"
else
    print_success "Go already installed: $(go version)"
fi

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
    print_success "Docker already installed: $(docker --version)"
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
    print_success "kubectl already installed: $(kubectl version --client)"
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
    print_success "Helm already installed: $(helm version --short)"
fi

# Install Go development tools
print_step "Installing Go development tools..."
if command -v go &> /dev/null; then
    # Install goimports
    if ! command -v goimports &> /dev/null; then
        go install golang.org/x/tools/cmd/goimports@latest
        print_success "goimports installed"
    else
        print_success "goimports already installed"
    fi
    
    # Install golangci-lint
    if ! command -v golangci-lint &> /dev/null; then
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
        print_success "golangci-lint installed"
    else
        print_success "golangci-lint already installed"
    fi
    
    # Install controller-gen
    if ! command -v controller-gen &> /dev/null; then
        go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
        print_success "controller-gen installed"
    else
        print_success "controller-gen already installed"
    fi
    
    # Install code-generator tools
    if [ ! -d "$(go env GOPATH)/src/k8s.io/code-generator" ]; then
        mkdir -p $(go env GOPATH)/src/k8s.io
        git clone https://github.com/kubernetes/code-generator.git $(go env GOPATH)/src/k8s.io/code-generator
        cd $(go env GOPATH)/src/k8s.io/code-generator
        go install ./cmd/...
        cd - > /dev/null
        print_success "code-generator tools installed"
    else
        print_success "code-generator tools already installed"
    fi
else
    print_error "Go not found, skipping Go development tools"
fi

# Install additional development tools
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

# Install kubectl plugins (optional)
fi
    
    # Docker extension
    code --install-extension ms-azuretools.vscode-docker
    
    # GitLens extension
    code --install-extension eamodio.gitlens
    
    print_success "VS Code extensions installed"
else
    print_warning "VS Code not found, skipping extensions installation"
fi

# Create Go workspace directory
print_step "Setting up Go workspace..."
mkdir -p ~/go/{bin,src,pkg}
if ! grep -q "GOPATH" ~/.bashrc; then
    echo 'export GOPATH=$HOME/go' >> ~/.bashrc
    echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
    export GOPATH=$HOME/go
    export PATH=$PATH:$GOPATH/bin
fi
print_success "Go workspace configured"

# Create .kube directory
print_step "Setting up kubectl configuration..."
mkdir -p ~/.kube
print_success "kubectl configuration directory created"

# Verify the installation
print_step "Verifying installation..."

# Check all installed tools
TOOLS=("go" "docker" "kubectl" "helm" "git" "make" "jq")
MISSING_TOOLS=()

for tool in "${TOOLS[@]}"; do
    if command -v "$tool" &> /dev/null; then
        print_success "$tool: $(command -v "$tool")"
    else
        print_error "$tool: Not found"
        MISSING_TOOLS+=("$tool")
    fi
done

if [ ${#MISSING_TOOLS[@]} -gt 0 ]; then
    print_warning "Some tools are missing: ${MISSING_TOOLS[*]}"
else
    print_success "All essential tools are installed"
fi

# Test Go installation
if command -v go &> /dev/null; then
    print_status "Testing Go installation..."
    go version
    go env GOPATH
    print_success "Go installation verified"
fi

# Test Docker installation
if command -v docker &> /dev/null; then
    print_status "Testing Docker installation..."
    docker --version
    docker run --rm hello-world > /dev/null 2>&1 && print_success "Docker installation verified" || print_warning "Docker installation may need user group setup"
fi

# Create a comprehensive summary file
SUMMARY_FILE="$HOME/kaiwo-poc-local-dev-setup-summary.txt"
cat > "$SUMMARY_FILE" << EOF
Kaiwo-PoC Local Development Environment Setup Summary
====================================================
Date: $(date)
Setup Script: kaiwo-PoC/scripts/setup-local-dev.sh

System Information:
- OS: $OS $VER
- Memory: ${MEMORY_GB}GB
- Disk Space: ${DISK_SPACE_GB}GB

Installed Components:
- Go (golang)
- Docker
- kubectl
- Helm
- Git
- Make
- jq
- goimports
- golangci-lint
- controller-gen
- code-generator tools
- k9s (Kubernetes CLI)
- kubectx/kubens
- kubectl tree plugin
- kubectl neat plugin
- VS Code extensions (if VS Code installed)

Go Environment:
- GOPATH: $(go env GOPATH 2>/dev/null || echo "Not set")
- GOROOT: $(go env GOROOT 2>/dev/null || echo "Not set")
- Go Version: $(go version 2>/dev/null || echo "Not available")

Docker Environment:
- Docker Version: $(docker --version 2>/dev/null || echo "Not available")
- Docker Group: $(groups | grep -q docker && echo "User in docker group" || echo "User not in docker group")

Next Steps for Kaiwo-PoC Development:
1. Navigate to project: cd ~/Desktop/kaiwo-PoC
2. Build project: make build
3. Run tests: make test
4. Generate code: make generate
5. Setup Kubernetes cluster: ./scripts/setup-complete-kubernetes.sh (if needed)

Useful Commands:
- Go to project: cd ~/Desktop/kaiwo-PoC
- Build: make build
- Test: make test
- Generate: make generate
- Lint: golangci-lint run
- Format: goimports -w .
- Docker build: make docker-build

Troubleshooting:
- If Docker permission denied: sudo usermod -aG docker \$USER (then log out/in)
- If Go tools not found: source ~/.bashrc
- If kubectl not working: check ~/.kube/config
EOF

print_success "Setup summary saved to: $SUMMARY_FILE"

# Create a quick reference card
REFERENCE_FILE="$HOME/kaiwo-poc-local-dev-quick-reference.txt"
cat > "$REFERENCE_FILE" << EOF
Kaiwo-PoC Local Development Quick Reference
==========================================

Project Navigation:
  cd ~/Desktop/kaiwo-PoC              # Go to project directory

Go Development:
  make build                          # Build the project
  make test                           # Run tests
  make generate                       # Generate code
  go mod tidy                         # Clean up dependencies
  goimports -w .                      # Format imports
  golangci-lint run                   # Run linter

Docker Operations:
  make docker-build                   # Build Docker image
  docker images                       # List images
  docker ps                           # List containers
  docker system prune                 # Clean up Docker

Kubernetes Operations:
  kubectl get nodes                   # View cluster nodes
  kubectl get pods --all-namespaces   # View all pods
  k9s                                 # Interactive cluster view
  kubectx                             # Switch contexts
  kubens                              # Switch namespaces

Development Tools:
  code .                              # Open VS Code in current directory
  git status                          # Check git status
  git add . && git commit -m "msg"    # Commit changes
  git push                            # Push changes

Troubleshooting:
  go version                          # Check Go version
  docker --version                    # Check Docker version
  kubectl version --client            # Check kubectl version
  echo \$GOPATH                       # Check GOPATH
  groups                              # Check user groups
EOF

print_success "Quick reference saved to: $REFERENCE_FILE"

# Display completion message
print_header "🎉 Local Development Setup Complete!"
print_success "Your Kaiwo-PoC local development environment is ready!"

echo
print_status "What's been installed:"
echo "  ✅ Go (golang) - Programming language"
echo "  ✅ Docker - Container runtime"
echo "  ✅ kubectl - Kubernetes CLI"
echo "  ✅ Helm - Package manager"
echo "  ✅ Git - Version control"
echo "  ✅ Make - Build automation"
echo "  ✅ jq - JSON processor"
echo "  ✅ goimports - Go import formatter"
echo "  ✅ golangci-lint - Go linter"
echo "  ✅ controller-gen - Code generator"
echo "  ✅ k9s - Interactive Kubernetes CLI"
echo "  ✅ kubectx/kubens - Context/namespace switching"
echo "  ✅ kubectl plugins - Tree and neat visualization"
echo "  ✅ VS Code extensions - Development IDE support"

echo
print_status "Next steps:"
echo "1. Navigate to your project:"
echo "   cd ~/Desktop/kaiwo-PoC"
echo
echo "2. Build the project:"
echo "   make build"
echo
echo "3. Run tests:"
echo "   make test"
echo
echo "4. Setup Kubernetes cluster (if needed):"
echo "   ./scripts/setup-complete-kubernetes.sh"
echo
echo "5. Check setup summary:"
echo "   cat $SUMMARY_FILE"
echo
echo "6. Quick reference:"
echo "   cat $REFERENCE_FILE"

echo
print_status "Important notes:"
echo "  🔄 You may need to log out and back in for Docker group changes to take effect"
echo "  🔄 Run 'source ~/.bashrc' to reload environment variables"
echo "  🔄 If you install VS Code later, run the script again to install extensions"

echo
print_status "Your local development environment is ready! 🚀"
print_status "Happy coding with Kaiwo-PoC! 🎯"
