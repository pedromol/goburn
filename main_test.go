package main

import (
	"os"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/fake"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected Config
	}{
		{
			name:    "default config",
			envVars: map[string]string{},
			expected: Config{
				TargetCPUUtilization:      80.0,
				TargetMemoryUtilization:   80.0,
				MinCPUUtilization:         20.0,
				MinMemoryUtilization:      20.0,
				MinNetworkUtilizationMbps: 20.0,
				MonitorInterval:           30 * time.Second,
				ScaleUpDelay:              60 * time.Second,
				ScaleDownDelay:            120 * time.Second,
				MaxMemoryMB:               1024,
				EnableMemoryUtilization:   true,
				NetworkInterface:          "eth0",
			},
		},
		{
			name: "custom config",
			envVars: map[string]string{
				"TARGET_CPU_UTILIZATION":       "90",
				"TARGET_MEMORY_UTILIZATION":    "85",
				"MIN_CPU_UTILIZATION":          "25",
				"MIN_MEMORY_UTILIZATION":       "30",
				"MIN_NETWORK_UTILIZATION_MBPS": "50",
				"MONITOR_INTERVAL_SECONDS":     "60",
				"SCALE_UP_DELAY_SECONDS":       "120",
				"SCALE_DOWN_DELAY_SECONDS":     "180",
				"MAX_MEMORY_MB":                "2048",
				"ENABLE_MEMORY_UTILIZATION":    "false",
				"NETWORK_INTERFACE":            "ens0",
				"NODE_NAME":                    "test-node",
			},
			expected: Config{
				TargetCPUUtilization:      90.0,
				TargetMemoryUtilization:   85.0,
				MinCPUUtilization:         25.0,
				MinMemoryUtilization:      30.0,
				MinNetworkUtilizationMbps: 50.0,
				MonitorInterval:           60 * time.Second,
				ScaleUpDelay:              120 * time.Second,
				ScaleDownDelay:            180 * time.Second,
				MaxMemoryMB:               2048,
				EnableMemoryUtilization:   false,
				NetworkInterface:          "ens0",
				NodeName:                  "test-node",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			config, err := loadConfig()
			if err != nil {
				t.Fatalf("loadConfig() error = %v", err)
			}

			// Check each field
			if config.TargetCPUUtilization != tt.expected.TargetCPUUtilization {
				t.Errorf("TargetCPUUtilization = %v, want %v", config.TargetCPUUtilization, tt.expected.TargetCPUUtilization)
			}
			if config.TargetMemoryUtilization != tt.expected.TargetMemoryUtilization {
				t.Errorf("TargetMemoryUtilization = %v, want %v", config.TargetMemoryUtilization, tt.expected.TargetMemoryUtilization)
			}
			if config.MinCPUUtilization != tt.expected.MinCPUUtilization {
				t.Errorf("MinCPUUtilization = %v, want %v", config.MinCPUUtilization, tt.expected.MinCPUUtilization)
			}
			if config.MinMemoryUtilization != tt.expected.MinMemoryUtilization {
				t.Errorf("MinMemoryUtilization = %v, want %v", config.MinMemoryUtilization, tt.expected.MinMemoryUtilization)
			}
			if config.MinNetworkUtilizationMbps != tt.expected.MinNetworkUtilizationMbps {
				t.Errorf("MinNetworkUtilizationMbps = %v, want %v", config.MinNetworkUtilizationMbps, tt.expected.MinNetworkUtilizationMbps)
			}
			if config.MonitorInterval != tt.expected.MonitorInterval {
				t.Errorf("MonitorInterval = %v, want %v", config.MonitorInterval, tt.expected.MonitorInterval)
			}
			if config.ScaleUpDelay != tt.expected.ScaleUpDelay {
				t.Errorf("ScaleUpDelay = %v, want %v", config.ScaleUpDelay, tt.expected.ScaleUpDelay)
			}
			if config.ScaleDownDelay != tt.expected.ScaleDownDelay {
				t.Errorf("ScaleDownDelay = %v, want %v", config.ScaleDownDelay, tt.expected.ScaleDownDelay)
			}
			if config.MaxMemoryMB != tt.expected.MaxMemoryMB {
				t.Errorf("MaxMemoryMB = %v, want %v", config.MaxMemoryMB, tt.expected.MaxMemoryMB)
			}
			if config.EnableMemoryUtilization != tt.expected.EnableMemoryUtilization {
				t.Errorf("EnableMemoryUtilization = %v, want %v", config.EnableMemoryUtilization, tt.expected.EnableMemoryUtilization)
			}
			if config.NetworkInterface != tt.expected.NetworkInterface {
				t.Errorf("NetworkInterface = %v, want %v", config.NetworkInterface, tt.expected.NetworkInterface)
			}
			if tt.expected.NodeName != "" && config.NodeName != tt.expected.NodeName {
				t.Errorf("NodeName = %v, want %v", config.NodeName, tt.expected.NodeName)
			}

			// Clean up environment variables
			for key := range tt.envVars {
				os.Unsetenv(key)
			}
		})
	}
}

func TestGetEnvFloat(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue float64
		expected     float64
	}{
		{
			name:         "valid float",
			key:          "TEST_FLOAT",
			value:        "3.14",
			defaultValue: 1.0,
			expected:     3.14,
		},
		{
			name:         "invalid float",
			key:          "TEST_FLOAT",
			value:        "invalid",
			defaultValue: 2.5,
			expected:     2.5,
		},
		{
			name:         "empty value",
			key:          "TEST_FLOAT",
			value:        "",
			defaultValue: 5.0,
			expected:     5.0,
		},
		{
			name:         "unset variable",
			key:          "UNSET_VAR",
			value:        "",
			defaultValue: 7.5,
			expected:     7.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvFloat(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvFloat(%s, %v) = %v, want %v", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue int
		expected     int
	}{
		{
			name:         "valid int",
			key:          "TEST_INT",
			value:        "42",
			defaultValue: 10,
			expected:     42,
		},
		{
			name:         "invalid int",
			key:          "TEST_INT",
			value:        "invalid",
			defaultValue: 25,
			expected:     25,
		},
		{
			name:         "empty value",
			key:          "TEST_INT",
			value:        "",
			defaultValue: 50,
			expected:     50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvInt(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvInt(%s, %v) = %v, want %v", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue bool
		expected     bool
	}{
		{
			name:         "true value",
			key:          "TEST_BOOL",
			value:        "true",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "false value",
			key:          "TEST_BOOL",
			value:        "false",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "1 value",
			key:          "TEST_BOOL",
			value:        "1",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "0 value",
			key:          "TEST_BOOL",
			value:        "0",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "invalid value",
			key:          "TEST_BOOL",
			value:        "invalid",
			defaultValue: true,
			expected:     true,
		},
		{
			name:         "empty value",
			key:          "TEST_BOOL",
			value:        "",
			defaultValue: false,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvBool(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvBool(%s, %v) = %v, want %v", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestGetEnvString(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue string
		expected     string
	}{
		{
			name:         "set value",
			key:          "TEST_STRING",
			value:        "hello",
			defaultValue: "default",
			expected:     "hello",
		},
		{
			name:         "empty value",
			key:          "TEST_STRING",
			value:        "",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "unset variable",
			key:          "UNSET_VAR",
			value:        "",
			defaultValue: "fallback",
			expected:     "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvString(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvString(%s, %s) = %s, want %s", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestRnd(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"length 0", 0},
		{"length 1", 1},
		{"length 10", 10},
		{"length 32", 32},
		{"length 100", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rnd(tt.length)
			if len(result) != tt.length {
				t.Errorf("rnd(%d) length = %d, want %d", tt.length, len(result), tt.length)
			}

			// Check that all characters are from the allowed set
			for _, char := range result {
				found := false
				for _, allowed := range l {
					if char == allowed {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("rnd(%d) contains invalid character: %c", tt.length, char)
				}
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		message string
	}{
		{
			name:    "simple message",
			key:     "12345678901234567890123456789012", // 32 bytes
			message: "hello world!!!!!",                 // 16 bytes (AES block size)
		},
		{
			name:    "empty message",
			key:     "abcdefghijklmnopqrstuvwxyz123456", // 32 bytes
			message: "0000000000000000",                 // 16 bytes
		},
		{
			name:    "numeric message",
			key:     "32109876543210987654321098765432", // 32 bytes
			message: "1234567890123456",                 // 16 bytes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted := encrypt(tt.key, tt.message)
			if encrypted == "" {
				t.Errorf("encrypt() returned empty string")
			}

			decrypted := decrypt(tt.key, encrypted)
			if decrypted != tt.message {
				t.Errorf("decrypt(encrypt(%s)) = %s, want %s", tt.message, decrypted, tt.message)
			}
		})
	}
}

func TestMinMax(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int64
		expected int64
	}{
		{"min: a < b", 5, 10, 5},
		{"min: a > b", 15, 8, 8},
		{"min: a == b", 7, 7, 7},
		{"min: negative", -5, -2, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name+" min", func(t *testing.T) {
			result := min(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}

	maxTests := []struct {
		name     string
		a, b     int64
		expected int64
	}{
		{"max: a < b", 5, 10, 10},
		{"max: a > b", 15, 8, 15},
		{"max: a == b", 7, 7, 7},
		{"max: negative", -5, -2, -2},
	}

	for _, tt := range maxTests {
		t.Run(tt.name+" max", func(t *testing.T) {
			result := max(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("max(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestMinInt(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"a < b", 5, 10, 5},
		{"a > b", 15, 8, 8},
		{"a == b", 7, 7, 7},
		{"negative", -5, -2, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := minInt(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("minInt(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"positive", 5.5, 5.5},
		{"negative", -3.2, 3.2},
		{"zero", 0.0, 0.0},
		{"large positive", 1000.123, 1000.123},
		{"large negative", -999.456, 999.456},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := abs(tt.input)
			if result != tt.expected {
				t.Errorf("abs(%f) = %f, want %f", tt.input, result, tt.expected)
			}
		})
	}
}

func createTestResourceBurner(t *testing.T) *ResourceBurner {
	config := Config{
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
	}

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

func TestResourceBurner_AddCPUSample(t *testing.T) {
	rb := createTestResourceBurner(t)

	// Test adding samples
	samples := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
	for _, sample := range samples {
		rb.addCPUSample(sample)
	}

	if len(rb.cpuSamples) != len(samples) {
		t.Errorf("Expected %d samples, got %d", len(samples), len(rb.cpuSamples))
	}

	for i, expected := range samples {
		if rb.cpuSamples[i] != expected {
			t.Errorf("Sample %d: expected %f, got %f", i, expected, rb.cpuSamples[i])
		}
	}
}

func TestResourceBurner_GetCPU95thPercentile(t *testing.T) {
	rb := createTestResourceBurner(t)

	// Test with no samples
	percentile := rb.getCPU95thPercentile()
	if percentile != 0 {
		t.Errorf("Expected 0 for empty samples, got %f", percentile)
	}

	// Test with known samples
	samples := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	for _, sample := range samples {
		rb.addCPUSample(sample)
	}

	percentile = rb.getCPU95thPercentile()
	// 95th percentile of [10,20,30,40,50,60,70,80,90,100] should be 100
	if percentile != 100 {
		t.Errorf("Expected 95th percentile to be 100, got %f", percentile)
	}

	// Test with more samples to verify percentile calculation
	rb.cpuSamples = []float64{} // Reset
	for i := 1; i <= 20; i++ {
		rb.addCPUSample(float64(i * 5)) // 5, 10, 15, ..., 100
	}

	percentile = rb.getCPU95thPercentile()
	// 95th percentile of 20 samples should be the 19th sample (95% of 20 = 19)
	expected := 95.0 // 19th sample in sequence 5,10,15,...,100
	if percentile != expected {
		t.Errorf("Expected 95th percentile to be %f, got %f", expected, percentile)
	}
}

func TestResourceBurner_AdjustCPULoad(t *testing.T) {
	rb := createTestResourceBurner(t)

	// Test scaling up
	rb.adjustCPULoad(80.0, 50.0) // target 80%, current 50%
	if rb.cpuWorkers <= 0 {
		t.Errorf("Expected CPU workers to be scaled up, got %d", rb.cpuWorkers)
	}

	initialWorkers := rb.cpuWorkers

	// Test scaling down (simulate high utilization)
	rb.adjustCPULoad(80.0, 95.0) // target 80%, current 95%
	if rb.cpuWorkers >= initialWorkers {
		t.Errorf("Expected CPU workers to be scaled down from %d, got %d", initialWorkers, rb.cpuWorkers)
	}
}

func TestResourceBurner_AdjustMemoryLoad(t *testing.T) {
	rb := createTestResourceBurner(t)

	initialMemory := len(rb.memoryData)

	// Test scaling up memory
	rb.adjustMemoryLoad(80.0, 50.0) // target 80%, current 50%
	if len(rb.memoryData) <= initialMemory {
		t.Errorf("Expected memory to be scaled up from %d bytes, got %d bytes", initialMemory, len(rb.memoryData))
	}

	currentMemory := len(rb.memoryData)

	// Test scaling down memory (simulate high utilization)
	rb.adjustMemoryLoad(80.0, 95.0) // target 80%, current 95%
	if len(rb.memoryData) >= currentMemory {
		t.Errorf("Expected memory to be scaled down from %d bytes, got %d bytes", currentMemory, len(rb.memoryData))
	}
}

func TestResourceBurner_AdjustNetworkLoad(t *testing.T) {
	rb := createTestResourceBurner(t)

	// Test scaling up
	rb.adjustNetworkLoad(30.0, 10.0) // target 30 Mbps, current 10 Mbps
	if rb.networkWorkers <= 0 {
		t.Errorf("Expected network workers to be scaled up, got %d", rb.networkWorkers)
	}

	initialWorkers := rb.networkWorkers

	// Test scaling down
	rb.adjustNetworkLoad(30.0, 40.0) // target 30 Mbps, current 40 Mbps
	if rb.networkWorkers >= initialWorkers {
		t.Errorf("Expected network workers to be scaled down from %d, got %d", initialWorkers, rb.networkWorkers)
	}
}

// Benchmark tests
func BenchmarkRnd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rnd(32)
	}
}

func BenchmarkEncryptDecrypt(b *testing.B) {
	key := "12345678901234567890123456789012"
	message := "hello world!!!!!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encrypted := encrypt(key, message)
		decrypt(key, encrypted)
	}
}

func BenchmarkCPUPercentileCalculation(b *testing.B) {
	rb := createTestResourceBurner(&testing.T{})

	// Fill with sample data
	for i := 0; i < 100; i++ {
		rb.addCPUSample(float64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.getCPU95thPercentile()
	}
}
