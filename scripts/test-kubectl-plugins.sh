#!/bin/bash

echo "🔍 Testing kubectl plugin installations..."

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Function to test kubectl plugin
test_plugin() {
    local plugin_name=$1
    local command=$2
    
    echo -n "Testing $plugin_name: "
    if $command --help &> /dev/null; then
        echo -e "${GREEN}✓ Installed${NC}"
        return 0
    else
        echo -e "${RED}✗ Not found${NC}"
        return 1
    fi
}

# Test kubectl tree plugin
test_plugin "kubectl tree" "kubectl tree"

# Test kubectl neat plugin
test_plugin "kubectl neat" "kubectl neat"

# Test other kubectl tools
echo
echo "Testing other kubectl tools:"
test_plugin "kubectx" "kubectx"
test_plugin "kubens" "kubens"

# Test k9s
echo
echo "Testing k9s:"
test_plugin "k9s" "k9s"

echo
echo "🔍 Plugin installation test complete!"
