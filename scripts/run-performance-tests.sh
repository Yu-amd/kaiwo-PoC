#!/bin/bash

# Performance Testing Script for Kaiwo
# This script runs comprehensive performance tests including realistic AI/ML workloads,
# profiling, load testing, and optimization validation.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TIMEOUT="5m"
VERBOSE=${VERBOSE:-false}
BENCHMARK_TIME="30s"
OUTPUT_DIR="performance-results"

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo -e "${BLUE}🚀 Kaiwo Performance Testing Suite${NC}"
echo -e "${BLUE}===================================${NC}"
echo ""

# Helper function to run tests with timing
run_test() {
    local test_name="$1"
    local test_command="$2"
    local start_time=$(date +%s)
    
    echo -e "${YELLOW}📊 Running: $test_name${NC}"
    echo "Command: $test_command"
    echo ""
    
    if $VERBOSE; then
        eval "$test_command" 2>&1 | tee "$OUTPUT_DIR/${test_name// /_}.log"
    else
        eval "$test_command" > "$OUTPUT_DIR/${test_name// /_}.log" 2>&1
    fi
    
    local exit_code=$?
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}✅ $test_name completed in ${duration}s${NC}"
    else
        echo -e "${RED}❌ $test_name failed after ${duration}s${NC}"
        if ! $VERBOSE; then
            echo "Error log:"
            tail -20 "$OUTPUT_DIR/${test_name// /_}.log"
        fi
    fi
    echo ""
    
    return $exit_code
}

# Helper function to run benchmarks
run_benchmark() {
    local bench_name="$1"
    local bench_command="$2"
    local start_time=$(date +%s)
    
    echo -e "${YELLOW}⚡ Benchmark: $bench_name${NC}"
    echo "Command: $bench_command"
    echo ""
    
    eval "$bench_command" 2>&1 | tee "$OUTPUT_DIR/${bench_name// /_}_benchmark.log"
    
    local exit_code=$?
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}⚡ $bench_name completed in ${duration}s${NC}"
        # Extract benchmark results
        echo "Benchmark Results:"
        grep -E "(ops/sec|MB|ns/op|goroutines)" "$OUTPUT_DIR/${bench_name// /_}_benchmark.log" | tail -5
    else
        echo -e "${RED}❌ $bench_name failed after ${duration}s${NC}"
    fi
    echo ""
    
    return $exit_code
}

echo -e "${BLUE}Phase 1: Unit Tests and Component Validation${NC}"
echo "============================================"

# Test 1: Performance Profiler
run_test "Performance Profiler Unit Test" \
    "cd /root/kaiwo-PoC && go test -v ./test/performance/profiling/ -run TestPerformanceProfiler -timeout $TIMEOUT"

# Test 2: Load Testing Framework
run_test "Load Testing Framework" \
    "cd /root/kaiwo-PoC && go test -v ./test/performance/load-testing/ -run TestLoadTestRunner_BaselineScenario -timeout $TIMEOUT"

echo -e "${BLUE}Phase 2: Performance Benchmarks${NC}"
echo "================================"

# Benchmark 1: Scheduling Performance
run_benchmark "Scheduling Performance" \
    "cd /root/kaiwo-PoC && go test -v ./test/performance/profiling/ -bench=BenchmarkSchedulingPerformanceWithProfiling -run=^$ -timeout $TIMEOUT -benchtime=$BENCHMARK_TIME"

# Benchmark 2: Memory Pressure Impact
run_benchmark "Memory Pressure Impact" \
    "cd /root/kaiwo-PoC && go test -v ./test/performance/profiling/ -bench=BenchmarkMemoryPressureImpact -run=^$ -timeout $TIMEOUT -benchtime=10s"

# Benchmark 3: Concurrent Scheduling
run_benchmark "Concurrent Scheduling" \
    "cd /root/kaiwo-PoC && go test -v ./test/performance/profiling/ -bench=BenchmarkConcurrentSchedulingWithContention -run=^$ -timeout $TIMEOUT -benchtime=10s"

# Benchmark 4: Resource Utilization Patterns
run_benchmark "Resource Utilization Patterns" \
    "cd /root/kaiwo-PoC && go test -v ./test/performance/profiling/ -bench=BenchmarkResourceUtilizationPatterns -run=^$ -timeout $TIMEOUT -benchtime=10s"

# Benchmark 5: Gang Scheduling Performance
run_benchmark "Gang Scheduling Performance" \
    "cd /root/kaiwo-PoC && go test -v ./test/performance/profiling/ -bench=BenchmarkGangSchedulingPerformance -run=^$ -timeout $TIMEOUT -benchtime=10s"

echo -e "${BLUE}Phase 3: Load Testing Scenarios${NC}"
echo "================================"

# Load Test 1: Stress Testing
run_test "Stress Test Scenario" \
    "cd /root/kaiwo-PoC && go test -v ./test/performance/load-testing/ -run TestLoadTestRunner_StressScenario -timeout $TIMEOUT"

# Load Test 2: GPU-Intensive Workloads
run_test "GPU-Intensive Workload Test" \
    "cd /root/kaiwo-PoC && go test -v ./test/performance/load-testing/ -run TestLoadTestRunner_GPUIntensiveScenario -timeout $TIMEOUT"

# Load Test 3: Burst Capacity
run_test "Burst Capacity Test" \
    "cd /root/kaiwo-PoC && go test -v ./test/performance/load-testing/ -run TestLoadTestRunner_BurstCapacityScenario -timeout $TIMEOUT"

echo -e "${BLUE}Phase 4: Realistic AI/ML Workload Simulation${NC}"
echo "============================================="

# AI/ML Benchmark: Realistic Workloads
run_benchmark "Realistic AI/ML Workloads" \
    "cd /root/kaiwo-PoC && go test -v ./test/performance/realistic-workloads/ -bench=BenchmarkRealisticAIMLWorkloads -run=^$ -timeout $TIMEOUT -benchtime=15s"

# AI/ML Benchmark: Gang Scheduling
run_benchmark "Gang Scheduling AI/ML Workloads" \
    "cd /root/kaiwo-PoC && go test -v ./test/performance/realistic-workloads/ -bench=BenchmarkGangSchedulingRealisticWorkloads -run=^$ -timeout $TIMEOUT -benchtime=10s"

# AI/ML Benchmark: Elastic Scaling
run_benchmark "Elastic Scaling AI/ML Workloads" \
    "cd /root/kaiwo-PoC && go test -v ./test/performance/realistic-workloads/ -bench=BenchmarkElasticScalingAIWorkloads -run=^$ -timeout $TIMEOUT -benchtime=10s"

echo -e "${BLUE}Phase 5: Performance Analysis and Reporting${NC}"
echo "==========================================="

# Generate performance summary
echo -e "${YELLOW}📋 Generating Performance Summary...${NC}"

cat > "$OUTPUT_DIR/performance_summary.md" << EOF
# Kaiwo Performance Test Results

**Test Date:** $(date)
**System:** $(uname -a)
**Go Version:** $(go version)

## Test Summary

EOF

# Count passed/failed tests
passed_tests=$(find "$OUTPUT_DIR" -name "*.log" -exec grep -l "PASS" {} \; | wc -l)
total_tests=$(find "$OUTPUT_DIR" -name "*.log" | wc -l)
failed_tests=$((total_tests - passed_tests))

cat >> "$OUTPUT_DIR/performance_summary.md" << EOF
- **Total Tests:** $total_tests
- **Passed:** $passed_tests
- **Failed:** $failed_tests
- **Success Rate:** $(( (passed_tests * 100) / total_tests ))%

## Detailed Results

EOF

# Add detailed results
for log_file in "$OUTPUT_DIR"/*.log; do
    if [ -f "$log_file" ]; then
        test_name=$(basename "$log_file" .log | tr '_' ' ')
        if grep -q "PASS" "$log_file"; then
            status="✅ PASSED"
        else
            status="❌ FAILED"
        fi
        
        echo "### $test_name - $status" >> "$OUTPUT_DIR/performance_summary.md"
        echo "" >> "$OUTPUT_DIR/performance_summary.md"
        
        # Extract key metrics if it's a benchmark
        if echo "$log_file" | grep -q "benchmark"; then
            echo "**Performance Metrics:**" >> "$OUTPUT_DIR/performance_summary.md"
            echo '```' >> "$OUTPUT_DIR/performance_summary.md"
            grep -E "(ops/sec|MB|ns/op|goroutines)" "$log_file" | tail -10 >> "$OUTPUT_DIR/performance_summary.md"
            echo '```' >> "$OUTPUT_DIR/performance_summary.md"
        fi
        
        echo "" >> "$OUTPUT_DIR/performance_summary.md"
    fi
done

echo -e "${GREEN}📊 Performance testing completed!${NC}"
echo -e "${GREEN}📁 Results saved to: $OUTPUT_DIR/${NC}"
echo -e "${GREEN}📋 Summary report: $OUTPUT_DIR/performance_summary.md${NC}"

# Display quick summary
echo ""
echo -e "${BLUE}Quick Summary:${NC}"
echo "=============="
echo "Total Tests: $total_tests"
echo "Passed: $passed_tests"
echo "Failed: $failed_tests"
echo "Success Rate: $(( (passed_tests * 100) / total_tests ))%"

if [ $failed_tests -gt 0 ]; then
    echo ""
    echo -e "${RED}Failed Tests:${NC}"
    for log_file in "$OUTPUT_DIR"/*.log; do
        if [ -f "$log_file" ] && ! grep -q "PASS" "$log_file"; then
            test_name=$(basename "$log_file" .log | tr '_' ' ')
            echo "  - $test_name"
        fi
    done
    exit 1
else
    echo ""
    echo -e "${GREEN}🎉 All performance tests passed!${NC}"
    exit 0
fi
