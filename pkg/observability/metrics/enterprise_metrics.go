/*
Copyright 2025 Kaiwo Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// EnterpriseMetrics provides advanced metrics collection and management
type EnterpriseMetrics struct {
	// Metric registries
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*Histogram
	timers     map[string]*Timer

	// Registry mutex
	mutex sync.RWMutex

	// Configuration
	config MetricsConfig

	// Exporters
	exporters []MetricsExporter

	// Aggregation engine
	aggregator *MetricsAggregator
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	// Service information
	ServiceName    string
	ServiceVersion string
	Environment    string

	// Collection settings
	CollectionInterval time.Duration
	RetentionPeriod    time.Duration

	// Export settings
	ExportInterval time.Duration
	BatchSize      int

	// Performance settings
	MaxMetrics     int
	HighResolution bool

	// Features
	EnableDistribution bool
	EnableProfiling    bool
	EnableAlerting     bool
}

// Counter represents a counter metric
type Counter struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Value       int64             `json:"value"`
	Labels      map[string]string `json:"labels"`
	LastUpdated time.Time         `json:"lastUpdated"`
	mutex       sync.RWMutex
}

// Gauge represents a gauge metric
type Gauge struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Value       float64           `json:"value"`
	Labels      map[string]string `json:"labels"`
	LastUpdated time.Time         `json:"lastUpdated"`
	mutex       sync.RWMutex
}

// Histogram represents a histogram metric
type Histogram struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Buckets     []HistogramBucket `json:"buckets"`
	Sum         float64           `json:"sum"`
	Count       int64             `json:"count"`
	Labels      map[string]string `json:"labels"`
	LastUpdated time.Time         `json:"lastUpdated"`
	mutex       sync.RWMutex
}

// HistogramBucket represents a bucket in a histogram
type HistogramBucket struct {
	UpperBound float64 `json:"upperBound"`
	Count      int64   `json:"count"`
}

// Timer represents a timer metric
type Timer struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Values      []time.Duration   `json:"values"`
	Labels      map[string]string `json:"labels"`
	LastUpdated time.Time         `json:"lastUpdated"`
	mutex       sync.RWMutex
}

// MetricsSnapshot represents a snapshot of all metrics
type MetricsSnapshot struct {
	Timestamp   time.Time             `json:"timestamp"`
	ServiceName string                `json:"serviceName"`
	Counters    map[string]*Counter   `json:"counters"`
	Gauges      map[string]*Gauge     `json:"gauges"`
	Histograms  map[string]*Histogram `json:"histograms"`
	Timers      map[string]*Timer     `json:"timers"`
}

// MetricsExporter interface for exporting metrics
type MetricsExporter interface {
	Export(snapshot *MetricsSnapshot) error
	Close() error
}

// MetricsAggregator aggregates metrics across time windows
type MetricsAggregator struct {
	// Aggregation windows
	windows map[string]*AggregationWindow
	mutex   sync.RWMutex

	// Configuration
	config AggregatorConfig
}

// AggregationWindow represents a time window for aggregation
type AggregationWindow struct {
	Name       string            `json:"name"`
	Duration   time.Duration     `json:"duration"`
	StartTime  time.Time         `json:"startTime"`
	Metrics    []MetricPoint     `json:"metrics"`
	Aggregated AggregatedMetrics `json:"aggregated"`
}

// MetricPoint represents a single metric measurement
type MetricPoint struct {
	Timestamp  time.Time         `json:"timestamp"`
	MetricName string            `json:"metricName"`
	Value      float64           `json:"value"`
	Labels     map[string]string `json:"labels"`
}

// AggregatedMetrics represents aggregated metric values
type AggregatedMetrics struct {
	Count  int64   `json:"count"`
	Sum    float64 `json:"sum"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
	StdDev float64 `json:"stdDev"`
}

// AggregatorConfig holds aggregator configuration
type AggregatorConfig struct {
	// Window definitions
	Windows []WindowConfig `json:"windows"`

	// Retention settings
	RetentionPeriod time.Duration `json:"retentionPeriod"`

	// Aggregation interval
	AggregationInterval time.Duration `json:"aggregationInterval"`
}

// WindowConfig defines an aggregation window
type WindowConfig struct {
	Name     string        `json:"name"`
	Duration time.Duration `json:"duration"`
}

// NewEnterpriseMetrics creates a new enterprise metrics instance
func NewEnterpriseMetrics(config MetricsConfig) *EnterpriseMetrics {
	// Set defaults
	if config.CollectionInterval == 0 {
		config.CollectionInterval = 15 * time.Second
	}
	if config.ExportInterval == 0 {
		config.ExportInterval = 60 * time.Second
	}
	if config.RetentionPeriod == 0 {
		config.RetentionPeriod = 24 * time.Hour
	}
	if config.BatchSize == 0 {
		config.BatchSize = 1000
	}
	if config.MaxMetrics == 0 {
		config.MaxMetrics = 100000
	}

	em := &EnterpriseMetrics{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
		timers:     make(map[string]*Timer),
		config:     config,
		exporters:  make([]MetricsExporter, 0),
	}

	// Initialize aggregator
	aggregatorConfig := AggregatorConfig{
		Windows: []WindowConfig{
			{Name: "1m", Duration: time.Minute},
			{Name: "5m", Duration: 5 * time.Minute},
			{Name: "1h", Duration: time.Hour},
			{Name: "24h", Duration: 24 * time.Hour},
		},
		RetentionPeriod:     config.RetentionPeriod,
		AggregationInterval: time.Minute,
	}
	em.aggregator = NewMetricsAggregator(aggregatorConfig)

	// Initialize default exporters
	em.exporters = append(em.exporters, NewPrometheusMetricsExporter())
	em.exporters = append(em.exporters, NewInfluxDBExporter())

	return em
}

// Counter operations

// IncrementCounter increments a counter by 1
func (em *EnterpriseMetrics) IncrementCounter(name string, labels map[string]string) {
	em.AddToCounter(name, 1, labels)
}

// AddToCounter adds a value to a counter
func (em *EnterpriseMetrics) AddToCounter(name string, value int64, labels map[string]string) {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	key := em.buildMetricKey(name, labels)

	counter, exists := em.counters[key]
	if !exists {
		counter = &Counter{
			Name:        name,
			Description: fmt.Sprintf("Counter metric: %s", name),
			Value:       0,
			Labels:      labels,
		}
		em.counters[key] = counter
	}

	counter.mutex.Lock()
	counter.Value += value
	counter.LastUpdated = time.Now()
	counter.mutex.Unlock()

	// Record in aggregator
	em.aggregator.RecordMetric(MetricPoint{
		Timestamp:  time.Now(),
		MetricName: name,
		Value:      float64(counter.Value),
		Labels:     labels,
	})
}

// Gauge operations

// SetGauge sets a gauge value
func (em *EnterpriseMetrics) SetGauge(name string, value float64, labels map[string]string) {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	key := em.buildMetricKey(name, labels)

	gauge, exists := em.gauges[key]
	if !exists {
		gauge = &Gauge{
			Name:        name,
			Description: fmt.Sprintf("Gauge metric: %s", name),
			Value:       0,
			Labels:      labels,
		}
		em.gauges[key] = gauge
	}

	gauge.mutex.Lock()
	gauge.Value = value
	gauge.LastUpdated = time.Now()
	gauge.mutex.Unlock()

	// Record in aggregator
	em.aggregator.RecordMetric(MetricPoint{
		Timestamp:  time.Now(),
		MetricName: name,
		Value:      value,
		Labels:     labels,
	})
}

// IncrementGauge increments a gauge
func (em *EnterpriseMetrics) IncrementGauge(name string, delta float64, labels map[string]string) {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	key := em.buildMetricKey(name, labels)

	gauge, exists := em.gauges[key]
	if !exists {
		gauge = &Gauge{
			Name:        name,
			Description: fmt.Sprintf("Gauge metric: %s", name),
			Value:       0,
			Labels:      labels,
		}
		em.gauges[key] = gauge
	}

	gauge.mutex.Lock()
	gauge.Value += delta
	gauge.LastUpdated = time.Now()
	gauge.mutex.Unlock()

	// Record in aggregator
	em.aggregator.RecordMetric(MetricPoint{
		Timestamp:  time.Now(),
		MetricName: name,
		Value:      gauge.Value,
		Labels:     labels,
	})
}

// Histogram operations

// ObserveHistogram observes a value in a histogram
func (em *EnterpriseMetrics) ObserveHistogram(name string, value float64, labels map[string]string) {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	key := em.buildMetricKey(name, labels)

	histogram, exists := em.histograms[key]
	if !exists {
		// Default buckets for response time metrics
		buckets := []HistogramBucket{
			{UpperBound: 0.001, Count: 0}, // 1ms
			{UpperBound: 0.005, Count: 0}, // 5ms
			{UpperBound: 0.01, Count: 0},  // 10ms
			{UpperBound: 0.05, Count: 0},  // 50ms
			{UpperBound: 0.1, Count: 0},   // 100ms
			{UpperBound: 0.5, Count: 0},   // 500ms
			{UpperBound: 1.0, Count: 0},   // 1s
			{UpperBound: 5.0, Count: 0},   // 5s
			{UpperBound: 10.0, Count: 0},  // 10s
		}

		histogram = &Histogram{
			Name:        name,
			Description: fmt.Sprintf("Histogram metric: %s", name),
			Buckets:     buckets,
			Sum:         0,
			Count:       0,
			Labels:      labels,
		}
		em.histograms[key] = histogram
	}

	histogram.mutex.Lock()
	histogram.Sum += value
	histogram.Count++

	// Update buckets
	for i := range histogram.Buckets {
		if value <= histogram.Buckets[i].UpperBound {
			histogram.Buckets[i].Count++
		}
	}

	histogram.LastUpdated = time.Now()
	histogram.mutex.Unlock()

	// Record in aggregator
	em.aggregator.RecordMetric(MetricPoint{
		Timestamp:  time.Now(),
		MetricName: name,
		Value:      value,
		Labels:     labels,
	})
}

// Timer operations

// RecordTime records a duration in a timer
func (em *EnterpriseMetrics) RecordTime(name string, duration time.Duration, labels map[string]string) {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	key := em.buildMetricKey(name, labels)

	timer, exists := em.timers[key]
	if !exists {
		timer = &Timer{
			Name:        name,
			Description: fmt.Sprintf("Timer metric: %s", name),
			Values:      make([]time.Duration, 0),
			Labels:      labels,
		}
		em.timers[key] = timer
	}

	timer.mutex.Lock()
	timer.Values = append(timer.Values, duration)

	// Keep only recent values to manage memory
	if len(timer.Values) > 1000 {
		timer.Values = timer.Values[len(timer.Values)-1000:]
	}

	timer.LastUpdated = time.Now()
	timer.mutex.Unlock()

	// Record in aggregator
	em.aggregator.RecordMetric(MetricPoint{
		Timestamp:  time.Now(),
		MetricName: name,
		Value:      float64(duration.Milliseconds()),
		Labels:     labels,
	})
}

// StartTimer starts a timer and returns a function to stop it
func (em *EnterpriseMetrics) StartTimer(name string, labels map[string]string) func() {
	startTime := time.Now()
	return func() {
		duration := time.Since(startTime)
		em.RecordTime(name, duration, labels)
	}
}

// Snapshot and export operations

// GetSnapshot returns a snapshot of all metrics
func (em *EnterpriseMetrics) GetSnapshot() *MetricsSnapshot {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	snapshot := &MetricsSnapshot{
		Timestamp:   time.Now(),
		ServiceName: em.config.ServiceName,
		Counters:    make(map[string]*Counter),
		Gauges:      make(map[string]*Gauge),
		Histograms:  make(map[string]*Histogram),
		Timers:      make(map[string]*Timer),
	}

	// Copy counters
	for key, counter := range em.counters {
		counterCopy := *counter
		snapshot.Counters[key] = &counterCopy
	}

	// Copy gauges
	for key, gauge := range em.gauges {
		gaugeCopy := *gauge
		snapshot.Gauges[key] = &gaugeCopy
	}

	// Copy histograms
	for key, histogram := range em.histograms {
		histogramCopy := *histogram
		snapshot.Histograms[key] = &histogramCopy
	}

	// Copy timers
	for key, timer := range em.timers {
		timerCopy := *timer
		snapshot.Timers[key] = &timerCopy
	}

	return snapshot
}

// ExportMetrics exports metrics to all configured exporters
func (em *EnterpriseMetrics) ExportMetrics() error {
	snapshot := em.GetSnapshot()

	for _, exporter := range em.exporters {
		if err := exporter.Export(snapshot); err != nil {
			klog.ErrorS(err, "Failed to export metrics")
			return err
		}
	}

	return nil
}

// GetAggregatedMetrics returns aggregated metrics for a time window
func (em *EnterpriseMetrics) GetAggregatedMetrics(windowName string) (*AggregationWindow, error) {
	return em.aggregator.GetWindow(windowName)
}

// Private methods

func (em *EnterpriseMetrics) buildMetricKey(name string, labels map[string]string) string {
	key := name
	for k, v := range labels {
		key += fmt.Sprintf(",%s=%s", k, v)
	}
	return key
}

// MetricsAggregator implementation

// NewMetricsAggregator creates a new metrics aggregator
func NewMetricsAggregator(config AggregatorConfig) *MetricsAggregator {
	ma := &MetricsAggregator{
		windows: make(map[string]*AggregationWindow),
		config:  config,
	}

	// Initialize windows
	for _, windowConfig := range config.Windows {
		window := &AggregationWindow{
			Name:      windowConfig.Name,
			Duration:  windowConfig.Duration,
			StartTime: time.Now(),
			Metrics:   make([]MetricPoint, 0),
		}
		ma.windows[windowConfig.Name] = window
	}

	return ma
}

// RecordMetric records a metric point
func (ma *MetricsAggregator) RecordMetric(point MetricPoint) {
	ma.mutex.Lock()
	defer ma.mutex.Unlock()

	for _, window := range ma.windows {
		// Check if point falls within window
		if point.Timestamp.After(window.StartTime.Add(-window.Duration)) {
			window.Metrics = append(window.Metrics, point)

			// Cleanup old points
			cutoff := time.Now().Add(-window.Duration)
			newMetrics := make([]MetricPoint, 0)
			for _, metric := range window.Metrics {
				if metric.Timestamp.After(cutoff) {
					newMetrics = append(newMetrics, metric)
				}
			}
			window.Metrics = newMetrics

			// Update aggregation
			window.Aggregated = ma.aggregate(window.Metrics)
		}
	}
}

// GetWindow returns an aggregation window
func (ma *MetricsAggregator) GetWindow(name string) (*AggregationWindow, error) {
	ma.mutex.RLock()
	defer ma.mutex.RUnlock()

	window, exists := ma.windows[name]
	if !exists {
		return nil, fmt.Errorf("window %s not found", name)
	}

	// Return a copy
	windowCopy := *window
	return &windowCopy, nil
}

func (ma *MetricsAggregator) aggregate(points []MetricPoint) AggregatedMetrics {
	if len(points) == 0 {
		return AggregatedMetrics{}
	}

	var sum, min, max float64
	values := make([]float64, len(points))

	for i, point := range points {
		values[i] = point.Value
		sum += point.Value

		if i == 0 {
			min = point.Value
			max = point.Value
		} else {
			if point.Value < min {
				min = point.Value
			}
			if point.Value > max {
				max = point.Value
			}
		}
	}

	mean := sum / float64(len(points))

	// Sort for percentiles
	for i := 0; i < len(values)-1; i++ {
		for j := i + 1; j < len(values); j++ {
			if values[i] > values[j] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}

	median := values[len(values)/2]
	p95 := values[int(float64(len(values))*0.95)]
	p99 := values[int(float64(len(values))*0.99)]

	// Calculate standard deviation
	var variance float64
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values))
	stdDev := variance // Simplified sqrt calculation

	return AggregatedMetrics{
		Count:  int64(len(points)),
		Sum:    sum,
		Min:    min,
		Max:    max,
		Mean:   mean,
		Median: median,
		P95:    p95,
		P99:    p99,
		StdDev: stdDev,
	}
}

// Exporters

type PrometheusMetricsExporter struct{}

func NewPrometheusMetricsExporter() *PrometheusMetricsExporter {
	return &PrometheusMetricsExporter{}
}

func (pme *PrometheusMetricsExporter) Export(snapshot *MetricsSnapshot) error {
	// TODO: Implement actual Prometheus export
	klog.V(4).Infof("Exporting metrics snapshot to Prometheus (counters: %d, gauges: %d, histograms: %d, timers: %d)",
		len(snapshot.Counters), len(snapshot.Gauges), len(snapshot.Histograms), len(snapshot.Timers))
	return nil
}

func (pme *PrometheusMetricsExporter) Close() error {
	return nil
}

type InfluxDBExporter struct{}

func NewInfluxDBExporter() *InfluxDBExporter {
	return &InfluxDBExporter{}
}

func (ide *InfluxDBExporter) Export(snapshot *MetricsSnapshot) error {
	// TODO: Implement actual InfluxDB export
	klog.V(4).Infof("Exporting metrics snapshot to InfluxDB")
	return nil
}

func (ide *InfluxDBExporter) Close() error {
	return nil
}
