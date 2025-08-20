#!/bin/bash

# Comprehensive Test Runner for Kaiwo Four-Phase Implementation
# This script runs all test phases to ensure no regressions

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CI_CONFIG="$PROJECT_ROOT/test/ci-config.yaml"
LOG_FILE="$PROJECT_ROOT/test-results.log"

# Test phases
PHASES=(
    "code-quality"
    "unit-tests"
    "integration-tests"
    "e2e-tests"
    "performance-tests"
    "security-tests"
    "docs-tests"
    "build-tests"
)

# Global variables
SKIP_PHASES=()
RUN_PHASES=()
VERBOSE=false
PARALLEL=false
DRY_RUN=false
FAIL_FAST=false

# Functions
log() {
    local level=$1
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    case $level in
        "INFO")
            echo -e "${BLUE}[INFO]${NC} $timestamp: $message" | tee -a "$LOG_FILE"
            ;;
        "SUCCESS")
            echo -e "${GREEN}[SUCCESS]${NC} $timestamp: $message" | tee -a "$LOG_FILE"
            ;;
        "WARNING")
            echo -e "${YELLOW}[WARNING]${NC} $timestamp: $message" | tee -a "$LOG_FILE"
            ;;
        "ERROR")
            echo -e "${RED}[ERROR]${NC} $timestamp: $message" | tee -a "$LOG_FILE"
            ;;
    esac
}

print_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Comprehensive test runner for Kaiwo four-phase implementation.

OPTIONS:
    -h, --help              Show this help message
    -v, --verbose           Enable verbose output
    -p, --parallel          Run tests in parallel where possible
    -d, --dry-run           Show what would be run without executing
    -f, --fail-fast         Stop on first failure
    --skip PHASE            Skip specific test phase(s)
    --only PHASE            Run only specific test phase(s)
    --config FILE           Use custom CI config file (default: test/ci-config.yaml)

TEST PHASES:
    code-quality           Static analysis, linting, formatting
    unit-tests            Unit tests with coverage
    integration-tests     Kubernetes integration tests
    e2e-tests             End-to-end tests
    performance-tests     Performance benchmarks
    security-tests        Security scanning
    docs-tests            Documentation validation
    build-tests           Build and package tests

EXAMPLES:
    $0                                    # Run all tests
    $0 --only unit-tests                  # Run only unit tests
    $0 --skip performance-tests           # Skip performance tests
    $0 --verbose --parallel               # Run with verbose output in parallel
    $0 --dry-run                          # Show what would be executed

EOF
}

check_prerequisites() {
    log "INFO" "Checking prerequisites..."
    
    # Check Go
    if ! command -v go &> /dev/null; then
        log "ERROR" "Go is not installed"
        exit 1
    fi
    
    # Check Python
    if ! command -v python3 &> /dev/null; then
        log "ERROR" "Python 3 is not installed"
        exit 1
    fi
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log "WARNING" "Docker is not installed - some tests may fail"
    fi
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log "WARNING" "kubectl is not installed - some tests may fail"
    fi
    
    # Check Kind
    if ! command -v kind &> /dev/null; then
        log "WARNING" "Kind is not installed - some tests may fail"
    fi
    
    log "SUCCESS" "Prerequisites check completed"
}

run_code_quality() {
    log "INFO" "Running code quality checks..."
    
    if [ "$DRY_RUN" = true ]; then
        log "INFO" "DRY RUN: Would run code quality checks"
        return 0
    fi
    
    cd "$PROJECT_ROOT"
    
    # Go code quality
    log "INFO" "Running Go linter..."
    make lint
    
    log "INFO" "Running Go vet..."
    go vet ./...
    
    log "INFO" "Checking Go formatting..."
    if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
        log "ERROR" "Code is not formatted. Please run 'go fmt ./...'"
        gofmt -s -l .
        return 1
    fi
    
    # Python code quality
    log "INFO" "Running Python linters..."
    cd python
    python -m pip install --upgrade pip
    pip install -r requirements-dev.txt
    
    flake8 kaiwo
    black --check kaiwo
    isort --check-only kaiwo
    
    log "INFO" "Running Python type checking..."
    mypy kaiwo --ignore-missing-imports
    
    cd "$PROJECT_ROOT"
    log "SUCCESS" "Code quality checks completed"
}

run_unit_tests() {
    log "INFO" "Running unit tests..."
    
    if [ "$DRY_RUN" = true ]; then
        log "INFO" "DRY RUN: Would run unit tests"
        return 0
    fi
    
    cd "$PROJECT_ROOT"
    
    # Go unit tests
    log "INFO" "Running Go unit tests..."
    make setup-envtest
    make test
    
    # Run with race detection and coverage
    go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
    
    # Python unit tests
    log "INFO" "Running Python unit tests..."
    cd python
    python -m pytest test/ -v --cov=kaiwo --cov-report=xml
    cd "$PROJECT_ROOT"
    
    log "SUCCESS" "Unit tests completed"
}

run_integration_tests() {
    log "INFO" "Running integration tests..."
    
    if [ "$DRY_RUN" = true ]; then
        log "INFO" "DRY RUN: Would run integration tests"
        return 0
    fi
    
    cd "$PROJECT_ROOT"
    
    # Check if Kind cluster exists
    if ! kind get clusters | grep -q "kaiwo-test"; then
        log "INFO" "Creating Kind cluster for integration tests..."
        test/setup_kind.sh
    fi
    
    # Install Chainsaw if not available
    if ! command -v chainsaw &> /dev/null; then
        log "INFO" "Installing Chainsaw..."
        curl -L https://github.com/kyverno/chainsaw/releases/latest/download/chainsaw_linux_amd64.tar.gz | tar -xz
        sudo mv chainsaw /usr/local/bin/
    fi
    
    # Run Chainsaw tests
    log "INFO" "Running Chainsaw integration tests..."
    chainsaw test --config test/chainsaw/configs/ci.yaml test/chainsaw/tests/standard/
    chainsaw test --config test/chainsaw/configs/ci.yaml test/chainsaw/tests/resource-requests/
    
    if [ -d "test/chainsaw/tests/sensitive" ]; then
        chainsaw test --config test/chainsaw/configs/ci.yaml test/chainsaw/tests/sensitive/
    fi
    
    if [ -d "test/chainsaw/tests-gpu" ]; then
        chainsaw test --config test/chainsaw/configs/ci.yaml test/chainsaw/tests-gpu/
    fi
    
    log "SUCCESS" "Integration tests completed"
}

run_e2e_tests() {
    log "INFO" "Running end-to-end tests..."
    
    if [ "$DRY_RUN" = true ]; then
        log "INFO" "DRY RUN: Would run e2e tests"
        return 0
    fi
    
    cd "$PROJECT_ROOT"
    
    # Check if Kind cluster exists
    if ! kind get clusters | grep -q "kaiwo-test"; then
        log "INFO" "Creating Kind cluster for e2e tests..."
        test/setup_kind.sh
    fi
    
    # Build and load operator image
    log "INFO" "Building operator image..."
    make docker-build IMG=ghcr.io/silogen/kaiwo-operator:v-e2e
    kind load docker-image ghcr.io/silogen/kaiwo-operator:v-e2e
    
    # Run e2e tests
    log "INFO" "Running e2e tests..."
    make test-e2e
    
    log "SUCCESS" "E2E tests completed"
}

run_performance_tests() {
    log "INFO" "Running performance tests..."
    
    if [ "$DRY_RUN" = true ]; then
        log "INFO" "DRY RUN: Would run performance tests"
        return 0
    fi
    
    cd "$PROJECT_ROOT"
    
    # Run performance benchmarks
    log "INFO" "Running performance benchmarks..."
    go test -v ./test/performance/ -bench=. -benchmem
    
    log "SUCCESS" "Performance tests completed"
}

run_security_tests() {
    log "INFO" "Running security tests..."
    
    if [ "$DRY_RUN" = true ]; then
        log "INFO" "DRY RUN: Would run security tests"
        return 0
    fi
    
    cd "$PROJECT_ROOT"
    
    # Install Trivy if not available
    if ! command -v trivy &> /dev/null; then
        log "INFO" "Installing Trivy..."
        curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin
    fi
    
    # Run Trivy vulnerability scan
    log "INFO" "Running Trivy vulnerability scan..."
    trivy fs --format json --output trivy-results.json .
    
    # Run Go security checks
    log "INFO" "Running Go security checks..."
    go install golang.org/x/vuln/cmd/govulncheck@latest
    govulncheck ./...
    
    # Check for secrets
    log "INFO" "Checking for secrets in code..."
    pip install detect-secrets
    detect-secrets scan --baseline .secrets.baseline || true
    
    log "SUCCESS" "Security tests completed"
}

run_docs_tests() {
    log "INFO" "Running documentation tests..."
    
    if [ "$DRY_RUN" = true ]; then
        log "INFO" "DRY RUN: Would run documentation tests"
        return 0
    fi
    
    cd "$PROJECT_ROOT"
    
    # Install documentation dependencies
    log "INFO" "Installing documentation dependencies..."
    cd docs
    pip install -r requirements.txt
    
    # Build documentation
    log "INFO" "Building documentation..."
    mkdocs build --strict
    
    # Validate YAML files
    log "INFO" "Validating YAML files..."
    find . -name "*.yaml" -o -name "*.yml" | xargs -I {} sh -c 'echo "Validating {}"; python -c "import yaml; yaml.safe_load(open(\"{}\"))"'
    
    cd "$PROJECT_ROOT"
    log "SUCCESS" "Documentation tests completed"
}

run_build_tests() {
    log "INFO" "Running build tests..."
    
    if [ "$DRY_RUN" = true ]; then
        log "INFO" "DRY RUN: Would run build tests"
        return 0
    fi
    
    cd "$PROJECT_ROOT"
    
    # Build CLI for multiple architectures
    log "INFO" "Building CLI for multiple architectures..."
    chmod +x build_cli_all_arch.sh
    ./build_cli_all_arch.sh "test-version"
    
    # Build Docker images
    log "INFO" "Building Docker images..."
    make docker-build IMG=ghcr.io/silogen/kaiwo-operator:test
    
    # Validate install manifests
    log "INFO" "Validating install manifests..."
    make build-installer IMG=ghcr.io/silogen/kaiwo-operator:test
    kubectl apply --dry-run=client -f dist/install.yaml
    
    log "SUCCESS" "Build tests completed"
}

run_phase() {
    local phase=$1
    
    case $phase in
        "code-quality")
            run_code_quality
            ;;
        "unit-tests")
            run_unit_tests
            ;;
        "integration-tests")
            run_integration_tests
            ;;
        "e2e-tests")
            run_e2e_tests
            ;;
        "performance-tests")
            run_performance_tests
            ;;
        "security-tests")
            run_security_tests
            ;;
        "docs-tests")
            run_docs_tests
            ;;
        "build-tests")
            run_build_tests
            ;;
        *)
            log "ERROR" "Unknown test phase: $phase"
            return 1
            ;;
    esac
}

main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                print_usage
                exit 0
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -p|--parallel)
                PARALLEL=true
                shift
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -f|--fail-fast)
                FAIL_FAST=true
                shift
                ;;
            --skip)
                SKIP_PHASES+=("$2")
                shift 2
                ;;
            --only)
                RUN_PHASES+=("$2")
                shift 2
                ;;
            --config)
                CI_CONFIG="$2"
                shift 2
                ;;
            *)
                log "ERROR" "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done
    
    # Initialize log file
    echo "Kaiwo Comprehensive Test Results - $(date)" > "$LOG_FILE"
    
    log "INFO" "Starting comprehensive test suite..."
    log "INFO" "Project root: $PROJECT_ROOT"
    log "INFO" "CI config: $CI_CONFIG"
    
    # Check prerequisites
    check_prerequisites
    
    # Determine which phases to run
    local phases_to_run=()
    if [ ${#RUN_PHASES[@]} -gt 0 ]; then
        phases_to_run=("${RUN_PHASES[@]}")
    else
        for phase in "${PHASES[@]}"; do
            if [[ ! " ${SKIP_PHASES[@]} " =~ " ${phase} " ]]; then
                phases_to_run+=("$phase")
            fi
        done
    fi
    
    log "INFO" "Phases to run: ${phases_to_run[*]}"
    
    # Run phases
    local failed_phases=()
    for phase in "${phases_to_run[@]}"; do
        log "INFO" "Starting phase: $phase"
        
        if run_phase "$phase"; then
            log "SUCCESS" "Phase $phase completed successfully"
        else
            log "ERROR" "Phase $phase failed"
            failed_phases+=("$phase")
            
            if [ "$FAIL_FAST" = true ]; then
                log "ERROR" "Stopping due to fail-fast mode"
                break
            fi
        fi
    done
    
    # Summary
    if [ ${#failed_phases[@]} -eq 0 ]; then
        log "SUCCESS" "All test phases completed successfully!"
        log "SUCCESS" "Ready for four-phase implementation roadmap!"
        exit 0
    else
        log "ERROR" "The following phases failed: ${failed_phases[*]}"
        log "ERROR" "Please fix the issues before proceeding with implementation"
        exit 1
    fi
}

# Run main function
main "$@"
