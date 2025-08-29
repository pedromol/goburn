package main

import (
	"time"

	"k8s.io/client-go/kubernetes/fake"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
)

// TestConfig provides test configurations for different scenarios
type TestConfig struct {
	Name        string
	Config      Config
	Description string
}

// GetTestConfigs returns predefined test configurations
func GetTestConfigs() []TestConfig {
	return []TestConfig{
		{
			Name: "default",
			Config: Config{
				TargetCPUUtilization:      80.0,
				TargetMemoryUtilization:   80.0,
				MinCPUUtilization:         20.0,
				MinMemoryUtilization:      20.0,
				MinNetworkUtilizationMbps: 20.0,
				MonitorInterval:           30 * time.Second,
				ScaleUpDelay:              60 * time.Second,
				ScaleDownDelay:            120 * time.Second,
				MaxMemoryMB:               1024,
				NodeName:                  "test-node",
				EnableMemoryUtilization:   true,
				NetworkInterface:          "eth0",
			},
			Description: "Default configuration with all features enabled",
		},
		{
			Name: "amd64_node",
			Config: Config{
				TargetCPUUtilization:      80.0,
				TargetMemoryUtilization:   80.0,
				MinCPUUtilization:         20.0,
				MinMemoryUtilization:      0.0, // No memory requirement for AMD64
				MinNetworkUtilizationMbps: 20.0,
				MonitorInterval:           30 * time.Second,
				ScaleUpDelay:              60 * time.Second,
				ScaleDownDelay:            120 * time.Second,
				MaxMemoryMB:               2048,
				NodeName:                  "amd64-node",
				EnableMemoryUtilization:   false, // Disabled for AMD64
				NetworkInterface:          "eth0",
			},
			Description: "AMD64 node configuration (CPU + Network only)",
		},
		{
			Name: "arm64_node",
			Config: Config{
				TargetCPUUtilization:      80.0,
				TargetMemoryUtilization:   80.0,
				MinCPUUtilization:         20.0,
				MinMemoryUtilization:      20.0, // Memory requirement for ARM64
				MinNetworkUtilizationMbps: 20.0,
				MonitorInterval:           30 * time.Second,
				ScaleUpDelay:              60 * time.Second,
				ScaleDownDelay:            120 * time.Second,
				MaxMemoryMB:               2048,
				NodeName:                  "arm64-node",
				EnableMemoryUtilization:   true, // Enabled for ARM64
				NetworkInterface:          "eth0",
			},
			Description: "ARM64 node configuration (CPU + Network + Memory)",
		},
		{
			Name: "aggressive",
			Config: Config{
				TargetCPUUtilization:      90.0,
				TargetMemoryUtilization:   85.0,
				MinCPUUtilization:         25.0,
				MinMemoryUtilization:      25.0,
				MinNetworkUtilizationMbps: 30.0,
				MonitorInterval:           15 * time.Second,
				ScaleUpDelay:              30 * time.Second,
				ScaleDownDelay:            60 * time.Second,
				MaxMemoryMB:               4096,
				NodeName:                  "aggressive-node",
				EnableMemoryUtilization:   true,
				NetworkInterface:          "eth0",
			},
			Description: "Aggressive configuration for maximum utilization",
		},
		{
			Name: "conservative",
			Config: Config{
				TargetCPUUtilization:      70.0,
				TargetMemoryUtilization:   75.0,
				MinCPUUtilization:         15.0,
				MinMemoryUtilization:      15.0,
				MinNetworkUtilizationMbps: 15.0,
				MonitorInterval:           60 * time.Second,
				ScaleUpDelay:              120 * time.Second,
				ScaleDownDelay:            300 * time.Second,
				MaxMemoryMB:               512,
				NodeName:                  "conservative-node",
				EnableMemoryUtilization:   true,
				NetworkInterface:          "eth0",
			},
			Description: "Conservative configuration for production environments",
		},
		{
			Name: "memory_disabled",
			Config: Config{
				TargetCPUUtilization:      80.0,
				TargetMemoryUtilization:   80.0,
				MinCPUUtilization:         20.0,
				MinMemoryUtilization:      0.0,
				MinNetworkUtilizationMbps: 20.0,
				MonitorInterval:           30 * time.Second,
				ScaleUpDelay:              60 * time.Second,
				ScaleDownDelay:            120 * time.Second,
				MaxMemoryMB:               1024,
				NodeName:                  "memory-disabled-node",
				EnableMemoryUtilization:   false,
				NetworkInterface:          "eth0",
			},
			Description: "Configuration with memory utilization disabled",
		},
	}
}

// CreateTestResourceBurnerWithConfig creates a ResourceBurner for testing with specific config
func CreateTestResourceBurnerWithConfig(config Config) *ResourceBurner {
	k8sClient := fake.NewSimpleClientset()
	metricsClient := metricsfake.NewSimpleClientset()

	return &ResourceBurner{
		config:           config,
		k8sClient:        k8sClient,
		metricsClient:    metricsClient,
		memoryData:       make([]byte, 0),
		cpuWorkers:       0,
		stopChannels:     make([]chan bool, 0),
		networkWorkers:   0,
		networkStopChans: make([]chan bool, 0),
		cpuSamples:       make([]float64, 0),
	}
}

// TestScenario represents a testing scenario with expected outcomes
type TestScenario struct {
	Name                  string
	Description           string
	CurrentCPU            float64
	CurrentMemory         float64
	CurrentNetwork        float64
	CPU95thPercentile     float64
	ExpectedCPUScale      string // "up", "down", "none"
	ExpectedMemoryScale   string // "up", "down", "none"
	ExpectedNetworkScale  string // "up", "down", "none"
	ShouldEnforceMinimums bool
}

// GetTestScenarios returns predefined test scenarios
func GetTestScenarios() []TestScenario {
	return []TestScenario{
		{
			Name:                  "all_below_minimum",
			Description:           "All metrics below minimum requirements",
			CurrentCPU:            15.0,
			CurrentMemory:         15.0,
			CurrentNetwork:        15.0,
			CPU95thPercentile:     15.0,
			ExpectedCPUScale:      "up",
			ExpectedMemoryScale:   "up",
			ExpectedNetworkScale:  "up",
			ShouldEnforceMinimums: true,
		},
		{
			Name:                  "all_above_minimum_below_target",
			Description:           "All metrics above minimum but below target",
			CurrentCPU:            25.0,
			CurrentMemory:         25.0,
			CurrentNetwork:        25.0,
			CPU95thPercentile:     25.0,
			ExpectedCPUScale:      "up",
			ExpectedMemoryScale:   "up",
			ExpectedNetworkScale:  "none",
			ShouldEnforceMinimums: false,
		},
		{
			Name:                  "all_at_target",
			Description:           "All metrics at target levels",
			CurrentCPU:            80.0,
			CurrentMemory:         80.0,
			CurrentNetwork:        20.0,
			CPU95thPercentile:     80.0,
			ExpectedCPUScale:      "none",
			ExpectedMemoryScale:   "none",
			ExpectedNetworkScale:  "none",
			ShouldEnforceMinimums: false,
		},
		{
			Name:                  "all_above_target",
			Description:           "All metrics above target levels",
			CurrentCPU:            95.0,
			CurrentMemory:         95.0,
			CurrentNetwork:        40.0,
			CPU95thPercentile:     95.0,
			ExpectedCPUScale:      "down",
			ExpectedMemoryScale:   "down",
			ExpectedNetworkScale:  "down",
			ShouldEnforceMinimums: false,
		},
		{
			Name:                  "mixed_scenario",
			Description:           "Mixed scenario: CPU low, Memory high, Network at minimum",
			CurrentCPU:            30.0,
			CurrentMemory:         90.0,
			CurrentNetwork:        20.0,
			CPU95thPercentile:     30.0,
			ExpectedCPUScale:      "up",
			ExpectedMemoryScale:   "down",
			ExpectedNetworkScale:  "none",
			ShouldEnforceMinimums: false,
		},
		{
			Name:                  "cpu_95th_below_minimum",
			Description:           "CPU 95th percentile below minimum despite current CPU being acceptable",
			CurrentCPU:            60.0,
			CurrentMemory:         60.0,
			CurrentNetwork:        25.0,
			CPU95thPercentile:     18.0, // Below 20% minimum
			ExpectedCPUScale:      "up",
			ExpectedMemoryScale:   "none",
			ExpectedNetworkScale:  "none",
			ShouldEnforceMinimums: true,
		},
	}
}

// TestUtilizationData represents utilization data for testing
type TestUtilizationData struct {
	CPUPercent     float64
	MemoryPercent  float64
	NetworkMbps    float64
	CPU95thPercent float64
}

// SimulateUtilizationHistory creates a history of utilization data for testing percentiles
func SimulateUtilizationHistory(samples int, baseValue, variance float64) []float64 {
	history := make([]float64, samples)
	for i := 0; i < samples; i++ {
		// Add some randomness to the base value
		variation := (float64(i%10) - 5) * variance / 5 // Simple variation pattern
		history[i] = baseValue + variation
		if history[i] < 0 {
			history[i] = 0
		}
		if history[i] > 100 {
			history[i] = 100
		}
	}
	return history
}
