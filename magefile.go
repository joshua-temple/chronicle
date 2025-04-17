//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Default target when running mage
var Default = All

// Aliases provides aliases for targets
var Aliases = map[string]interface{}{
	"t": Test,
	"b": Build,
	"e": Examples,
}

// All runs the lint, test, and build targets
func All() error {
	mg.Deps(Lint, Test, Build)
	return nil
}

// Build builds the project
func Build() error {
	fmt.Println("Building Chronicle...")
	return sh.Run("go", "build", "-v", "./...")
}

// Test runs all tests
func Test() error {
	fmt.Println("Running tests...")
	return sh.Run("go", "test", "-v", "./...")
}

// TestShort runs tests with short flag (skips integration tests)
func TestShort() error {
	fmt.Println("Running short tests...")
	return sh.Run("go", "test", "-v", "-short", "./...")
}

// Examples runs only the example tests
func Examples() error {
	fmt.Println("Running example tests...")
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return sh.Run("bash", "-c", fmt.Sprintf("cd %s/examples && go test -v ./...", pwd))
}

// Clean removes build artifacts
func Clean() error {
	fmt.Println("Cleaning build artifacts...")
	if err := os.RemoveAll("bin"); err != nil {
		return err
	}
	return sh.Run("go", "clean")
}

// Format formats code
func Format() error {
	fmt.Println("Formatting code...")
	return sh.Run("go", "fmt", "./...")
}

// Lint lints code
func Lint() error {
	fmt.Println("Linting code...")
	return sh.Run("golangci-lint", "run", "./...")
}

// Doc generates documentation
func Doc() error {
	fmt.Println("Generating documentation...")
	return sh.Run("go", "doc", "-all", "./...")
}

// Coverage runs tests with coverage
func Coverage() error {
	fmt.Println("Running tests with coverage...")
	if err := sh.Run("go", "test", "-coverprofile=coverage.out", "./..."); err != nil {
		return err
	}
	return sh.Run("go", "tool", "cover", "-html=coverage.out", "-o", "coverage.html")
}

// StopInfra stops any running infrastructure from tests
func StopInfra() error {
	fmt.Println("Stopping test infrastructure...")
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return sh.Run("bash", "-c", fmt.Sprintf("cd %s/examples && docker-compose down -v", pwd))
}

// Docker builds Docker image with test environment
func Docker() error {
	fmt.Println("Building Docker image...")
	return sh.Run("docker", "build", "-t", "chronicle-test", "-f", "Dockerfile.test", ".")
}

// Install installs Mage if not already installed
func Install() error {
	fmt.Println("Checking if Mage is installed...")
	_, err := exec.LookPath("mage")
	if err == nil {
		fmt.Println("Mage is already installed")
		return nil
	}

	fmt.Println("Installing Mage...")
	return sh.Run("go", "install", "github.com/magefile/mage@latest")
}

// Help prints mage help
func Help() error {
	fmt.Println("Chronicle Magefile targets:")
	fmt.Println("  mage [target]")
	fmt.Println("")
	fmt.Println("Targets:")
	fmt.Println("  all          - Default target: lint, test, and build")
	fmt.Println("  build (b)    - Build the project")
	fmt.Println("  test (t)     - Run all tests")
	fmt.Println("  testShort    - Run tests with -short flag (skips integration tests)")
	fmt.Println("  examples (e) - Run only the example tests")
	fmt.Println("  clean        - Clean build artifacts")
	fmt.Println("  format       - Format code")
	fmt.Println("  lint         - Lint code")
	fmt.Println("  doc          - Generate documentation")
	fmt.Println("  coverage     - Run tests with coverage")
	fmt.Println("  stopInfra    - Stop any running infrastructure from tests")
	fmt.Println("  docker       - Build Docker image with test environment")
	fmt.Println("  install      - Install Mage if not already installed")
	fmt.Println("  help         - Show this help message")

	return nil
}
