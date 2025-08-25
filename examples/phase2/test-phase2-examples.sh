#!/bin/bash

# Kaiwo Phase 2 Examples Test Runner
# Tests all Phase 2 examples to ensure they work correctly

set -e

echo "🧪 Testing Kaiwo Phase 2 Examples"
echo "=================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_TIMEOUT=30
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run a test
run_test() {
    local test_name=$1
    local test_file=$2
    
    echo -e "${BLUE}🧪 Testing: ${test_name}${NC}"
    echo "   File: $test_file"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    # Apply the example
    if kubectl apply -f "$test_file" --dry-run=client &>/dev/null; then
        echo -e "${GREEN}✅ YAML validation passed${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}❌ YAML validation failed${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
    
    echo ""
}

echo -e "${BLUE}🔍 Checking prerequisites...${NC}"

# Check kubectl
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}❌ kubectl not found${NC}"
    exit 1
fi

# Check cluster connectivity
if ! kubectl cluster-info &> /dev/null; then
    echo -e "${RED}❌ Cannot connect to cluster${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Prerequisites met${NC}"
echo ""

echo -e "${YELLOW}==================== TESTING GANG SCHEDULING ====================${NC}"
echo ""

# Test gang scheduling examples
run_test "Distributed Training Gang" "gang-scheduling/01-distributed-training-gang.yaml"
run_test "Multi-Node Inference Gang" "gang-scheduling/02-multi-node-inference-gang.yaml"
run_test "Research Cluster Gang" "gang-scheduling/03-research-cluster-gang.yaml"

echo -e "${YELLOW}==================== TESTING ELASTIC SCALING ====================${NC}"
echo ""

# Test elastic scaling examples
run_test "Web Service Elastic" "elastic-scaling/01-web-service-elastic.yaml"
run_test "Batch Processing Elastic" "elastic-scaling/02-batch-processing-elastic.yaml"
run_test "Training Elastic" "elastic-scaling/03-training-elastic.yaml"

echo -e "${YELLOW}==================== TESTING ADVANCED FEATURES ==================${NC}"
echo ""

# Test advanced features
run_test "Gang + Elastic Hybrid" "advanced-features/01-gang-elastic-hybrid.yaml"
run_test "Multi-Model Pipeline" "advanced-features/02-multi-model-pipeline.yaml"
run_test "Research Federation" "advanced-features/03-research-federation.yaml"

echo ""
echo -e "${BLUE}📊 Test Results Summary${NC}"
echo "========================="
echo -e "${GREEN}✅ Passed: $PASSED_TESTS${NC}"
echo -e "${RED}❌ Failed: $FAILED_TESTS${NC}"
echo -e "${BLUE}📈 Total:  $TOTAL_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo ""
    echo -e "${GREEN}🎉 All Phase 2 examples passed validation!${NC}"
    echo ""
    echo -e "${YELLOW}💡 Next steps:${NC}"
    echo "1. Apply examples: ./apply-all-examples.sh"
    echo "2. Run demo: ./demo-phase2-features.sh"
    echo "3. Monitor workloads: kubectl get kaiwojobs --all-namespaces -w"
    echo ""
    exit 0
else
    echo ""
    echo -e "${RED}❌ Some tests failed. Please check the YAML files.${NC}"
    exit 1
fi
