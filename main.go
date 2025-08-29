package main

import (
	"bufio"
	"context"
	"crypto/aes"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Config struct {
	TargetCPUUtilization      float64
	TargetMemoryUtilization   float64
	MinCPUUtilization         float64
	MinMemoryUtilization      float64
	MinNetworkUtilizationMbps float64
	MonitorInterval           time.Duration
	ScaleUpDelay              time.Duration
	ScaleDownDelay            time.Duration
	MaxMemoryMB               int64
	NodeName                  string
	EnableMemoryUtilization   bool
	NetworkInterface          string
}

type ResourceBurner struct {
	config        Config
	k8sClient     kubernetes.Interface
	metricsClient metricsclientset.Interface

	// Resource control
	memoryData       []byte
	memoryMutex      sync.RWMutex
	cpuWorkers       int
	cpuMutex         sync.RWMutex
	stopChannels     []chan bool
	networkWorkers   int
	networkMutex     sync.RWMutex
	networkStopChans []chan bool

	// State tracking
	lastScaleAction time.Time
	scalingUp       bool

	// CPU percentile tracking
	cpuSamples     []float64
	cpuSampleMutex sync.RWMutex
}

var l = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func rnd(n int) string {
	s := make([]rune, n)
	for i := range s {
		s[i] = l[rand.Intn(len(l))]
	}
	return string(s)
}

func encrypt(k string, m string) string {
	c, _ := aes.NewCipher([]byte(k))
	msg := make([]byte, len(m))
	c.Encrypt(msg, []byte(m))
	return hex.EncodeToString(msg)
}

func decrypt(k string, m string) string {
	txt, _ := hex.DecodeString(m)
	c, _ := aes.NewCipher([]byte(k))
	msg := make([]byte, len(txt))
	c.Decrypt(msg, txt)
	return string(msg)
}

func NewResourceBurner() (*ResourceBurner, error) {
	config, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}

	// Create in-cluster config
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s config: %v", err)
	}

	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %v", err)
	}

	metricsClient, err := metricsclientset.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics client: %v", err)
	}

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
	}, nil
}

func loadConfig() (Config, error) {
	config := Config{
		TargetCPUUtilization:      getEnvFloat("TARGET_CPU_UTILIZATION", 80.0),
		TargetMemoryUtilization:   getEnvFloat("TARGET_MEMORY_UTILIZATION", 80.0),
		MinCPUUtilization:         getEnvFloat("MIN_CPU_UTILIZATION", 20.0),
		MinMemoryUtilization:      getEnvFloat("MIN_MEMORY_UTILIZATION", 20.0),
		MinNetworkUtilizationMbps: getEnvFloat("MIN_NETWORK_UTILIZATION_MBPS", 20.0),
		MonitorInterval:           time.Duration(getEnvInt("MONITOR_INTERVAL_SECONDS", 30)) * time.Second,
		ScaleUpDelay:              time.Duration(getEnvInt("SCALE_UP_DELAY_SECONDS", 60)) * time.Second,
		ScaleDownDelay:            time.Duration(getEnvInt("SCALE_DOWN_DELAY_SECONDS", 120)) * time.Second,
		MaxMemoryMB:               int64(getEnvInt("MAX_MEMORY_MB", 1024)),
		NodeName:                  os.Getenv("NODE_NAME"),
		EnableMemoryUtilization:   getEnvBool("ENABLE_MEMORY_UTILIZATION", true),
		NetworkInterface:          getEnvString("NETWORK_INTERFACE", "eth0"),
	}

	if config.NodeName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return config, fmt.Errorf("failed to get hostname and NODE_NAME not set: %v", err)
		}
		config.NodeName = hostname
	}

	return config, nil
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvString(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (rb *ResourceBurner) getNodeMetrics(ctx context.Context) (*metricsv1beta1.NodeMetrics, error) {
	nodeMetrics, err := rb.metricsClient.MetricsV1beta1().NodeMetricses().Get(ctx, rb.config.NodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node metrics: %v", err)
	}
	return nodeMetrics, nil
}

func (rb *ResourceBurner) getCurrentUtilization(ctx context.Context) (cpuPercent, memoryPercent float64, err error) {
	nodeMetrics, err := rb.getNodeMetrics(ctx)
	if err != nil {
		return 0, 0, err
	}

	// Get node capacity
	node, err := rb.k8sClient.CoreV1().Nodes().Get(ctx, rb.config.NodeName, metav1.GetOptions{})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get node info: %v", err)
	}

	cpuCapacity := node.Status.Capacity.Cpu().MilliValue()
	memoryCapacity := node.Status.Capacity.Memory().Value()

	cpuUsage := nodeMetrics.Usage.Cpu().MilliValue()
	memoryUsage := nodeMetrics.Usage.Memory().Value()

	cpuPercent = float64(cpuUsage) / float64(cpuCapacity) * 100
	memoryPercent = float64(memoryUsage) / float64(memoryCapacity) * 100

	return cpuPercent, memoryPercent, nil
}

func (rb *ResourceBurner) adjustCPULoad(targetUtilization, currentUtilization float64) {
	rb.cpuMutex.Lock()
	defer rb.cpuMutex.Unlock()

	utilizationDiff := targetUtilization - currentUtilization
	maxWorkers := runtime.NumCPU() * 2

	if utilizationDiff > 10 && rb.cpuWorkers < maxWorkers {
		// Scale up CPU workers
		newWorkers := minInt(int(utilizationDiff/20), maxWorkers-rb.cpuWorkers)
		for i := 0; i < newWorkers; i++ {
			stopChan := make(chan bool, 1)
			rb.stopChannels = append(rb.stopChannels, stopChan)
			go rb.cpuWorker(stopChan)
			rb.cpuWorkers++
		}
		log.Printf("Scaled up CPU workers to %d (utilization: %.1f%%, target: %.1f%%)",
			rb.cpuWorkers, currentUtilization, targetUtilization)

	} else if utilizationDiff < -10 && rb.cpuWorkers > 0 {
		// Scale down CPU workers
		workersToStop := minInt(rb.cpuWorkers, int(-utilizationDiff/20)+1)
		for i := 0; i < workersToStop && len(rb.stopChannels) > 0; i++ {
			// Stop the last worker
			lastIdx := len(rb.stopChannels) - 1
			rb.stopChannels[lastIdx] <- true
			rb.stopChannels = rb.stopChannels[:lastIdx]
			rb.cpuWorkers--
		}
		log.Printf("Scaled down CPU workers to %d (utilization: %.1f%%, target: %.1f%%)",
			rb.cpuWorkers, currentUtilization, targetUtilization)
	}
}

func (rb *ResourceBurner) adjustMemoryLoad(targetUtilization, currentUtilization float64) {
	rb.memoryMutex.Lock()
	defer rb.memoryMutex.Unlock()

	utilizationDiff := targetUtilization - currentUtilization

	if utilizationDiff > 10 {
		// Scale up memory usage
		currentSizeMB := int64(len(rb.memoryData) / 1024 / 1024)
		additionalMB := int64(utilizationDiff * 10) // Rough estimation
		newSizeMB := min(currentSizeMB+additionalMB, rb.config.MaxMemoryMB)

		if newSizeMB > currentSizeMB {
			newData := make([]byte, newSizeMB*1024*1024)
			copy(newData, rb.memoryData)

			// Fill new memory with random data
			for i := len(rb.memoryData); i < len(newData); i++ {
				newData[i] = byte(rand.Intn(256))
			}

			rb.memoryData = newData
			log.Printf("Scaled up memory to %d MB (utilization: %.1f%%, target: %.1f%%)",
				newSizeMB, currentUtilization, targetUtilization)
		}

	} else if utilizationDiff < -10 && len(rb.memoryData) > 0 {
		// Scale down memory usage
		currentSizeMB := int64(len(rb.memoryData) / 1024 / 1024)
		reductionMB := int64(-utilizationDiff * 10) // Rough estimation
		newSizeMB := max(0, currentSizeMB-reductionMB)

		if newSizeMB < currentSizeMB {
			if newSizeMB == 0 {
				rb.memoryData = make([]byte, 0)
			} else {
				rb.memoryData = rb.memoryData[:newSizeMB*1024*1024]
			}
			log.Printf("Scaled down memory to %d MB (utilization: %.1f%%, target: %.1f%%)",
				newSizeMB, currentUtilization, targetUtilization)
		}
	}
}

func (rb *ResourceBurner) cpuWorker(stopChan chan bool) {
	for {
		select {
		case <-stopChan:
			return
		default:
			// CPU intensive work
			key := rnd(32)
			decrypt(key, encrypt(key, key))
		}
	}
}

// Network worker to generate network traffic
func (rb *ResourceBurner) networkWorker(stopChan chan bool) {
	for {
		select {
		case <-stopChan:
			return
		default:
			// Generate network traffic by creating connections and sending data
			rb.generateNetworkTraffic()
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (rb *ResourceBurner) generateNetworkTraffic() {
	// Create a local connection to generate network stats
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Read and discard data
		buffer := make([]byte, 1024)
		conn.Read(buffer)
	}()

	// Connect and send data
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		return
	}
	defer conn.Close()

	// Send random data to generate network utilization
	data := make([]byte, 1024*10) // 10KB
	rand.Read(data)
	conn.Write(data)
}

func (rb *ResourceBurner) getNetworkUtilization() (float64, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, fmt.Errorf("failed to open /proc/net/dev: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var totalBytes int64

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, rb.config.NetworkInterface) {
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				rxBytes, _ := strconv.ParseInt(fields[1], 10, 64)
				txBytes, _ := strconv.ParseInt(fields[9], 10, 64)
				totalBytes = rxBytes + txBytes
			}
			break
		}
	}

	// Convert to Mbps (rough estimation)
	// This is a simplified calculation - in production you'd want to track over time
	mbps := float64(totalBytes) / (1024 * 1024) / 60 // Rough estimate per minute
	return mbps, nil
}

// CPU percentile tracking functions
func (rb *ResourceBurner) addCPUSample(cpuPercent float64) {
	rb.cpuSampleMutex.Lock()
	defer rb.cpuSampleMutex.Unlock()

	rb.cpuSamples = append(rb.cpuSamples, cpuPercent)

	// Keep only last 100 samples (about 50 minutes with 30s intervals)
	if len(rb.cpuSamples) > 100 {
		rb.cpuSamples = rb.cpuSamples[1:]
	}
}

func (rb *ResourceBurner) getCPU95thPercentile() float64 {
	rb.cpuSampleMutex.RLock()
	defer rb.cpuSampleMutex.RUnlock()

	if len(rb.cpuSamples) == 0 {
		return 0
	}

	// Copy samples to avoid modifying original
	samples := make([]float64, len(rb.cpuSamples))
	copy(samples, rb.cpuSamples)

	sort.Float64s(samples)

	// Calculate 95th percentile
	index := int(math.Ceil(0.95*float64(len(samples)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(samples) {
		index = len(samples) - 1
	}

	return samples[index]
}

func (rb *ResourceBurner) adjustNetworkLoad(targetMbps, currentMbps float64) {
	rb.networkMutex.Lock()
	defer rb.networkMutex.Unlock()

	utilizationDiff := targetMbps - currentMbps
	maxWorkers := 5 // Limit network workers to prevent overwhelming the system

	if utilizationDiff > 5 && rb.networkWorkers < maxWorkers {
		// Scale up network workers
		newWorkers := minInt(int(utilizationDiff/10)+1, maxWorkers-rb.networkWorkers)
		for i := 0; i < newWorkers; i++ {
			stopChan := make(chan bool, 1)
			rb.networkStopChans = append(rb.networkStopChans, stopChan)
			go rb.networkWorker(stopChan)
			rb.networkWorkers++
		}
		log.Printf("Scaled up network workers to %d (utilization: %.1f Mbps, target: %.1f Mbps)",
			rb.networkWorkers, currentMbps, targetMbps)

	} else if utilizationDiff < -5 && rb.networkWorkers > 0 {
		// Scale down network workers
		workersToStop := minInt(rb.networkWorkers, int(-utilizationDiff/10)+1)
		for i := 0; i < workersToStop && len(rb.networkStopChans) > 0; i++ {
			lastIdx := len(rb.networkStopChans) - 1
			rb.networkStopChans[lastIdx] <- true
			rb.networkStopChans = rb.networkStopChans[:lastIdx]
			rb.networkWorkers--
		}
		log.Printf("Scaled down network workers to %d (utilization: %.1f Mbps, target: %.1f Mbps)",
			rb.networkWorkers, currentMbps, targetMbps)
	}
}

func (rb *ResourceBurner) memoryWorker() {
	for {
		rb.memoryMutex.RLock()
		if len(rb.memoryData) > 0 {
			// Touch memory to prevent swapping
			for i := 0; i < len(rb.memoryData); i += 4096 {
				rb.memoryData[i] = byte(rand.Intn(256))
			}
		}
		rb.memoryMutex.RUnlock()
		time.Sleep(1 * time.Second)
	}
}

func (rb *ResourceBurner) monitor(ctx context.Context) {
	ticker := time.NewTicker(rb.config.MonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cpuUtil, memUtil, err := rb.getCurrentUtilization(ctx)
			if err != nil {
				log.Printf("Failed to get utilization metrics: %v", err)
				continue
			}

			// Add CPU sample for percentile tracking
			rb.addCPUSample(cpuUtil)
			cpu95th := rb.getCPU95thPercentile()

			// Get network utilization
			networkUtil, _ := rb.getNetworkUtilization()

			log.Printf("Current utilization - CPU: %.1f%% (95th: %.1f%%), Memory: %.1f%%, Network: %.1f Mbps, Workers: %d/%d, Memory: %d MB",
				cpuUtil, cpu95th, memUtil, networkUtil, rb.cpuWorkers, rb.networkWorkers, len(rb.memoryData)/1024/1024)

			// Only adjust if enough time has passed since last scaling action
			now := time.Now()
			if rb.scalingUp && now.Sub(rb.lastScaleAction) < rb.config.ScaleUpDelay {
				continue
			}
			if !rb.scalingUp && now.Sub(rb.lastScaleAction) < rb.config.ScaleDownDelay {
				continue
			}

			// ENFORCE MINIMUM REQUIREMENTS FIRST
			needsMinimumEnforcement := false

			// 1. CPU 95th percentile must be > 20%
			if cpu95th < rb.config.MinCPUUtilization {
				log.Printf("⚠️  CPU 95th percentile (%.1f%%) below minimum requirement (%.1f%%) - scaling up",
					cpu95th, rb.config.MinCPUUtilization)
				rb.adjustCPULoad(rb.config.MinCPUUtilization+10, cpuUtil) // Add buffer
				needsMinimumEnforcement = true
			}

			// 2. Memory utilization must be > 20% (for nodes where enabled)
			if rb.config.EnableMemoryUtilization && memUtil < rb.config.MinMemoryUtilization {
				log.Printf("⚠️  Memory utilization (%.1f%%) below minimum requirement (%.1f%%) - scaling up",
					memUtil, rb.config.MinMemoryUtilization)
				rb.adjustMemoryLoad(rb.config.MinMemoryUtilization+10, memUtil) // Add buffer
				needsMinimumEnforcement = true
			}

			// 3. Network utilization must be > 20%
			if networkUtil < rb.config.MinNetworkUtilizationMbps {
				log.Printf("⚠️  Network utilization (%.1f Mbps) below minimum requirement (%.1f Mbps) - scaling up",
					networkUtil, rb.config.MinNetworkUtilizationMbps)
				rb.adjustNetworkLoad(rb.config.MinNetworkUtilizationMbps+5, networkUtil) // Add buffer
				needsMinimumEnforcement = true
			}

			// If we're enforcing minimums, skip normal target-based adjustments
			if needsMinimumEnforcement {
				rb.lastScaleAction = now
				rb.scalingUp = true
				continue
			}

			// NORMAL TARGET-BASED ADJUSTMENTS (only if minimums are met)
			needsCPUAdjustment := abs(cpuUtil-rb.config.TargetCPUUtilization) > 10
			needsMemoryAdjustment := rb.config.EnableMemoryUtilization && abs(memUtil-rb.config.TargetMemoryUtilization) > 10
			needsNetworkAdjustment := abs(networkUtil-rb.config.MinNetworkUtilizationMbps) > 5

			if needsCPUAdjustment || needsMemoryAdjustment || needsNetworkAdjustment {
				rb.scalingUp = cpuUtil < rb.config.TargetCPUUtilization ||
					memUtil < rb.config.TargetMemoryUtilization ||
					networkUtil < rb.config.MinNetworkUtilizationMbps
				rb.lastScaleAction = now

				if needsCPUAdjustment {
					rb.adjustCPULoad(rb.config.TargetCPUUtilization, cpuUtil)
				}
				if needsMemoryAdjustment {
					rb.adjustMemoryLoad(rb.config.TargetMemoryUtilization, memUtil)
				}
				if needsNetworkAdjustment {
					rb.adjustNetworkLoad(rb.config.MinNetworkUtilizationMbps, networkUtil)
				}
			}
		}
	}
}

func (rb *ResourceBurner) Run(ctx context.Context) error {
	log.Printf("🔥 Starting dynamic resource burner on node %s", rb.config.NodeName)
	log.Printf("📊 Target utilization - CPU: %.1f%%, Memory: %.1f%%",
		rb.config.TargetCPUUtilization, rb.config.TargetMemoryUtilization)
	log.Printf("⚠️  MINIMUM REQUIREMENTS - CPU 95th percentile: >%.1f%%, Memory: >%.1f%%, Network: >%.1f Mbps",
		rb.config.MinCPUUtilization, rb.config.MinMemoryUtilization, rb.config.MinNetworkUtilizationMbps)
	log.Printf("🌐 Network interface: %s, Memory utilization enabled: %v",
		rb.config.NetworkInterface, rb.config.EnableMemoryUtilization)

	// Start memory worker
	go rb.memoryWorker()

	// Start monitoring
	rb.monitor(ctx)

	return nil
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func main() {
	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	burner, err := NewResourceBurner()
	if err != nil {
		log.Fatalf("Failed to create resource burner: %v", err)
	}

	// Start resource burner in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- burner.Run(ctx)
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		log.Printf("🛑 Received shutdown signal, gracefully stopping...")
		cancel()

		// Give some time for graceful cleanup
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		// Stop CPU workers gracefully
		burner.cpuMutex.Lock()
		log.Printf("Stopping %d CPU workers...", len(burner.stopChannels))
		for _, stopChan := range burner.stopChannels {
			select {
			case stopChan <- true:
			case <-shutdownCtx.Done():
			}
		}
		burner.stopChannels = nil
		burner.cpuWorkers = 0
		burner.cpuMutex.Unlock()

		// Stop network workers gracefully
		burner.networkMutex.Lock()
		log.Printf("Stopping %d network workers...", len(burner.networkStopChans))
		for _, stopChan := range burner.networkStopChans {
			select {
			case stopChan <- true:
			case <-shutdownCtx.Done():
			}
		}
		burner.networkStopChans = nil
		burner.networkWorkers = 0
		burner.networkMutex.Unlock()

		// Release memory
		burner.memoryMutex.Lock()
		log.Printf("Releasing %d MB of allocated memory...", len(burner.memoryData)/1024/1024)
		burner.memoryData = nil
		burner.memoryMutex.Unlock()

		log.Printf("✅ Graceful shutdown completed")

	case err := <-errChan:
		if err != nil {
			log.Fatalf("Resource burner failed: %v", err)
		}
	}
}
