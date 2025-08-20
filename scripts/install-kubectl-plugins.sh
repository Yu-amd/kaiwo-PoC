#!/bin/bash

echo "🔧 Installing kubectl plugins manually..."

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# Create local bin directory for kubectl plugins
LOCAL_BIN="$HOME/.local/bin"
mkdir -p "$LOCAL_BIN"

# Add to PATH if not already there
if [[ ":$PATH:" != *":$LOCAL_BIN:"* ]]; then
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
    export PATH="$HOME/.local/bin:$PATH"
    print_status "Added $LOCAL_BIN to PATH"
fi

# Function to download and install kubectl plugin
install_plugin() {
    local plugin_name=$1
    local download_url=$2
    local binary_name=$3
    
    print_status "Installing $plugin_name plugin..."
    
    # Create temporary directory
    local temp_dir=$(mktemp -d)
    cd "$temp_dir"
    
    # Download the plugin
    if curl -L -o "$binary_name" "$download_url" 2>/dev/null; then
        # Make it executable
        chmod +x "$binary_name"
        
        # Move to local bin directory
        mv "$binary_name" "$LOCAL_BIN/"
        
        print_success "$plugin_name plugin installed to $LOCAL_BIN"
        cd - > /dev/null
        rm -rf "$temp_dir"
        return 0
    else
        print_error "Failed to download $plugin_name plugin"
        cd - > /dev/null
        rm -rf "$temp_dir"
        return 1
    fi
}

# Try to install kubectl tree plugin
print_status "Attempting to install kubectl tree plugin..."
if install_plugin "kubectl tree" \
    "https://github.com/ahmetb/kubectl-tree/releases/download/v0.4.3/kubectl-tree_linux_amd64" \
    "kubectl-tree"; then
    TREE_INSTALLED=true
else
    print_warning "kubectl tree plugin installation failed - trying alternative method..."
    # Try alternative URL
    if install_plugin "kubectl tree" \
        "https://github.com/ahmetb/kubectl-tree/releases/latest/download/kubectl-tree_linux_amd64" \
        "kubectl-tree"; then
        TREE_INSTALLED=true
    else
        TREE_INSTALLED=false
    fi
fi

# Try to install kubectl neat plugin
print_status "Attempting to install kubectl neat plugin..."
if install_plugin "kubectl neat" \
    "https://github.com/itaysk/kubectl-neat/releases/download/v1.2.0/kubectl-neat_linux_amd64" \
    "kubectl-neat"; then
    NEAT_INSTALLED=true
else
    print_warning "kubectl neat plugin installation failed - trying alternative method..."
    # Try alternative URL
    if install_plugin "kubectl neat" \
        "https://github.com/itaysk/kubectl-neat/releases/latest/download/kubectl-neat_linux_amd64" \
        "kubectl-neat"; then
        NEAT_INSTALLED=true
    else
        NEAT_INSTALLED=false
    fi
fi

# Test installations
echo
print_status "Testing installations..."

# Reload PATH for current session
export PATH="$HOME/.local/bin:$PATH"

if [ "$TREE_INSTALLED" = true ] && kubectl tree --help &> /dev/null; then
    print_success "kubectl tree plugin is working"
else
    print_warning "kubectl tree plugin is not working"
fi

if [ "$NEAT_INSTALLED" = true ] && kubectl neat --help &> /dev/null; then
    print_success "kubectl neat plugin is working"
else
    print_warning "kubectl neat plugin is not working"
fi

echo
print_status "Plugin installation complete!"
print_status "Note: These plugins are optional and not required for Kaiwo-PoC development."
print_status "If installation failed, you can still use kubectl without these plugins."
print_status "You may need to restart your terminal or run 'source ~/.bashrc' for PATH changes to take effect."
