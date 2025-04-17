#!/bin/bash

# Run Chronicle integration tests
# This script provides utilities for running individual tests or the entire suite using Mage

set -e

# Default values
SINGLE_TEST=false
CLEAN_INFRA=false
TEST_PATH=""
TEST_TIMEOUT=2m

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    -s|--single)
      SINGLE_TEST=true
      TEST_PATH="$2"
      shift
      shift
      ;;
    -c|--clean)
      CLEAN_INFRA=true
      shift
      ;;
    -t|--timeout)
      TEST_TIMEOUT="$2"
      shift
      shift
      ;;
    *)
      echo "Unknown option: $key"
      exit 1
      ;;
  esac
done

# Ensure Mage is installed
command -v mage >/dev/null 2>&1 || { 
  echo "Installing Mage..."
  go install github.com/magefile/mage@latest
}

# Function to clean up infrastructure
cleanup_infrastructure() {
  echo "Cleaning up infrastructure..."
  mage stopInfra
}

# Clean infrastructure if requested
if $CLEAN_INFRA; then
  cleanup_infrastructure
fi

# Set up environment variables
export CHRONICLE_TIMEOUT="$TEST_TIMEOUT"

if $SINGLE_TEST; then
  # Run a single test
  if [[ -z "$TEST_PATH" ]]; then
    echo "Error: Test path must be specified for single test mode"
    exit 1
  fi
  
  echo "Running single test: $TEST_PATH with timeout $TEST_TIMEOUT"
  cd "$(dirname "$TEST_PATH")" && go test -v "$(basename "$TEST_PATH")" -timeout "$TEST_TIMEOUT"
else
  # Run all tests
  echo "Running all tests with timeout $TEST_TIMEOUT"
  mage test
fi

# Cleanup on exit if running in CI environment
if [[ -n "$CI" ]]; then
  cleanup_infrastructure
fi

echo "Done!"
