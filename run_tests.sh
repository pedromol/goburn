#!/bin/bash

# Comprehensive test runner for goburn
# Usage: ./run_tests.sh [unit|integration|benchmark|coverage|all]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_TIMEOUT=${TEST_TIMEOUT:-300s}
VERBOSE=${VERBOSE:-false}
COVERAGE_THRESHOLD=${COVERAGE_THRESHOLD:-80}

print_header() {
    echo -e "${BLUE}=================================================${NC}"
    echo -e "${BLUE}🔥 goburn Test Suite${NC}"
    echo -e "${BLUE}=================================================${NC}"
    echo ""
}

print_section() {
    echo -e "${YELLOW}📋 $1${NC}"
    echo "----------------------------------------"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# Function to run unit tests
run_unit_tests() {
    print_section "Running Unit Tests"
    
    if [ "$VERBOSE" = "true" ]; then
        go test -v -timeout $TEST_TIMEOUT -run "^Test[^I]" ./...
    else
        go test -timeout $TEST_TIMEOUT -run "^Test[^I]" ./...
    fi
    
    if [ $? -eq 0 ]; then
        print_success "Unit tests passed"
    else
        print_error "Unit tests failed"
        exit 1
    fi
    echo ""
}

# Function to run integration tests
run_integration_tests() {
    print_section "Running Integration Tests"
    
    if [ "$VERBOSE" = "true" ]; then
        go test -v -timeout $TEST_TIMEOUT -run "^TestI" ./...
    else
        go test -timeout $TEST_TIMEOUT -run "^TestI" ./...
    fi
    
    if [ $? -eq 0 ]; then
        print_success "Integration tests passed"
    else
        print_error "Integration tests failed"
        exit 1
    fi
    echo ""
}

# Function to run benchmark tests
run_benchmark_tests() {
    print_section "Running Benchmark Tests"
    
    echo "🏃 CPU Performance Benchmarks:"
    go test -bench=BenchmarkRnd -benchmem -count=3
    echo ""
    
    echo "🔐 Encryption Performance Benchmarks:"
    go test -bench=BenchmarkEncryptDecrypt -benchmem -count=3
    echo ""
    
    echo "📊 CPU Percentile Calculation Benchmarks:"
    go test -bench=BenchmarkCPUPercentileCalculation -benchmem -count=3
    echo ""
    
    echo "⚖️  Resource Adjustment Benchmarks:"
    go test -bench=BenchmarkResourceBurner -benchmem -count=3
    echo ""
    
    print_success "Benchmark tests completed"
    echo ""
}

# Function to run tests with coverage
run_coverage_tests() {
    print_section "Running Tests with Coverage Analysis"
    
    # Create coverage directory
    mkdir -p coverage
    
    # Run tests with coverage
    go test -timeout $TEST_TIMEOUT -coverprofile=coverage/coverage.out -covermode=atomic ./...
    
    if [ $? -eq 0 ]; then
        # Generate coverage report
        go tool cover -html=coverage/coverage.out -o coverage/coverage.html
        
        # Get coverage percentage
        COVERAGE=$(go tool cover -func=coverage/coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        
        echo ""
        echo "📊 Coverage Report:"
        go tool cover -func=coverage/coverage.out | tail -10
        echo ""
        
        if (( $(echo "$COVERAGE >= $COVERAGE_THRESHOLD" | bc -l) )); then
            print_success "Coverage: ${COVERAGE}% (meets threshold of ${COVERAGE_THRESHOLD}%)"
            print_success "HTML coverage report generated: coverage/coverage.html"
        else
            print_warning "Coverage: ${COVERAGE}% (below threshold of ${COVERAGE_THRESHOLD}%)"
        fi
    else
        print_error "Coverage tests failed"
        exit 1
    fi
    echo ""
}

# Function to run race condition tests
run_race_tests() {
    print_section "Running Race Condition Tests"
    
    go test -race -timeout $TEST_TIMEOUT ./...
    
    if [ $? -eq 0 ]; then
        print_success "Race condition tests passed"
    else
        print_error "Race condition tests failed"
        exit 1
    fi
    echo ""
}

# Function to run memory leak tests
run_memory_tests() {
    print_section "Running Memory Leak Tests"
    
    # Test for memory leaks in CPU workers
    echo "🧠 Testing CPU worker memory usage:"
    go test -run TestResourceBurner_AdjustCPULoad -memprofile=coverage/cpu_mem.prof
    
    # Test for memory leaks in memory allocation
    echo "💾 Testing memory allocation patterns:"
    go test -run TestResourceBurner_AdjustMemoryLoad -memprofile=coverage/memory_mem.prof
    
    print_success "Memory tests completed (profiles saved in coverage/)"
    echo ""
}

# Function to validate test configurations
validate_test_configs() {
    print_section "Validating Test Configurations"
    
    go test -run TestLoadConfig -v
    go test -run TestConfigValidation -v
    
    if [ $? -eq 0 ]; then
        print_success "Test configuration validation passed"
    else
        print_error "Test configuration validation failed"
        exit 1
    fi
    echo ""
}

# Function to test architecture-specific scenarios
test_architecture_scenarios() {
    print_section "Testing Architecture-Specific Scenarios"
    
    echo "🖥️  Testing AMD64 configuration (no memory requirement):"
    ENABLE_MEMORY_UTILIZATION=false MIN_MEMORY_UTILIZATION=0 go test -run TestResourceBurner_ScalingBehavior -v
    
    echo ""
    echo "💪 Testing ARM64 configuration (with memory requirement):"
    ENABLE_MEMORY_UTILIZATION=true MIN_MEMORY_UTILIZATION=20 go test -run TestResourceBurner_ScalingBehavior -v
    
    if [ $? -eq 0 ]; then
        print_success "Architecture-specific tests passed"
    else
        print_error "Architecture-specific tests failed"
        exit 1
    fi
    echo ""
}

# Function to run stress tests
run_stress_tests() {
    print_section "Running Stress Tests"
    
    echo "🔥 Stress testing with high worker counts:"
    go test -run TestResourceBurner_WorkerLimits -timeout 60s
    
    echo "📈 Stress testing percentile calculations with large datasets:"
    go test -run TestResourceBurner_CPUSampleManagement -timeout 60s
    
    if [ $? -eq 0 ]; then
        print_success "Stress tests passed"
    else
        print_error "Stress tests failed"
        exit 1
    fi
    echo ""
}

# Function to check for common issues
check_code_quality() {
    print_section "Code Quality Checks"
    
    # Check if go fmt is needed
    echo "🎨 Checking code formatting:"
    UNFORMATTED=$(gofmt -l *.go)
    if [ -n "$UNFORMATTED" ]; then
        print_warning "The following files need formatting:"
        echo "$UNFORMATTED"
        echo "Run: go fmt ./..."
    else
        print_success "Code formatting is correct"
    fi
    
    # Check if go vet passes
    echo ""
    echo "🔍 Running go vet:"
    go vet ./...
    if [ $? -eq 0 ]; then
        print_success "go vet passed"
    else
        print_error "go vet found issues"
        exit 1
    fi
    
    # Check for common security issues (if gosec is available)
    if command -v gosec &> /dev/null; then
        echo ""
        echo "🔒 Running security checks:"
        gosec ./...
        if [ $? -eq 0 ]; then
            print_success "Security checks passed"
        else
            print_warning "Security issues found (review gosec output)"
        fi
    fi
    
    echo ""
}

# Function to run all tests
run_all_tests() {
    print_header
    
    # Ensure dependencies are up to date
    echo "📦 Ensuring dependencies are up to date:"
    go mod tidy
    go mod download
    echo ""
    
    check_code_quality
    validate_test_configs
    run_unit_tests
    run_integration_tests
    test_architecture_scenarios
    run_race_tests
    run_memory_tests
    run_stress_tests
    run_coverage_tests
    run_benchmark_tests
    
    print_section "Test Summary"
    print_success "All tests completed successfully! 🎉"
    echo ""
    echo "📊 Generated Reports:"
    echo "   - Coverage: coverage/coverage.html"
    echo "   - Memory profiles: coverage/*.prof"
    echo ""
    echo "💡 Next Steps:"
    echo "   - Review coverage report for areas needing more tests"
    echo "   - Check memory profiles if performance issues are suspected"
    echo "   - Run './deploy.sh status' to test deployment"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  unit          Run unit tests only"
    echo "  integration   Run integration tests only"
    echo "  benchmark     Run benchmark tests only"
    echo "  coverage      Run tests with coverage analysis"
    echo "  race          Run race condition tests"
    echo "  memory        Run memory leak tests"
    echo "  stress        Run stress tests"
    echo "  quality       Run code quality checks"
    echo "  arch          Test architecture-specific scenarios"
    echo "  all           Run all tests (default)"
    echo "  help          Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  TEST_TIMEOUT           Test timeout (default: 300s)"
    echo "  VERBOSE                Verbose output (default: false)"
    echo "  COVERAGE_THRESHOLD     Coverage threshold % (default: 80)"
    echo ""
    echo "Examples:"
    echo "  $0 unit                    # Run unit tests only"
    echo "  VERBOSE=true $0 coverage   # Run coverage tests with verbose output"
    echo "  TEST_TIMEOUT=600s $0 all   # Run all tests with 10-minute timeout"
}

# Main script logic
case "${1:-all}" in
    "unit")
        print_header
        run_unit_tests
        ;;
    "integration")
        print_header
        run_integration_tests
        ;;
    "benchmark")
        print_header
        run_benchmark_tests
        ;;
    "coverage")
        print_header
        run_coverage_tests
        ;;
    "race")
        print_header
        run_race_tests
        ;;
    "memory")
        print_header
        run_memory_tests
        ;;
    "stress")
        print_header
        run_stress_tests
        ;;
    "quality")
        print_header
        check_code_quality
        ;;
    "arch")
        print_header
        test_architecture_scenarios
        ;;
    "all")
        run_all_tests
        ;;
    "help"|"-h"|"--help")
        show_usage
        ;;
    *)
        echo "❌ Unknown command: $1"
        echo ""
        show_usage
        exit 1
        ;;
esac
