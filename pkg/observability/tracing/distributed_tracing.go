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

package tracing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// TracingManager provides enterprise-grade distributed tracing
type TracingManager struct {
	// Active traces
	traces     map[string]*Trace
	traceMutex sync.RWMutex

	// Sampling configuration
	samplingConfig SamplingConfig

	// Exporters
	exporters []TraceExporter

	// Configuration
	config TracingConfig
}

// TracingConfig holds tracing configuration
type TracingConfig struct {
	// Service information
	ServiceName    string
	ServiceVersion string
	Environment    string

	// Sampling rate (0.0 to 1.0)
	SamplingRate float64

	// Batch configuration
	BatchTimeout time.Duration
	BatchSize    int

	// Resource limits
	MaxTraces   int
	MaxSpanSize int

	// Enable specific features
	EnableMetrics   bool
	EnableLogging   bool
	EnableProfiling bool
}

// SamplingConfig defines sampling behavior
type SamplingConfig struct {
	// Default sampling rate
	DefaultRate float64

	// Per-operation sampling rates
	OperationRates map[string]float64

	// Rate limiting
	RateLimit int

	// Adaptive sampling
	AdaptiveSampling bool
}

// Trace represents a distributed trace
type Trace struct {
	// Trace identification
	TraceID  string `json:"traceID"`
	ParentID string `json:"parentID,omitempty"`

	// Trace metadata
	ServiceName   string            `json:"serviceName"`
	OperationName string            `json:"operationName"`
	Tags          map[string]string `json:"tags"`

	// Timing
	StartTime time.Time     `json:"startTime"`
	Duration  time.Duration `json:"duration"`

	// Spans
	Spans []Span `json:"spans"`

	// Status
	Status TraceStatus `json:"status"`
	Error  string      `json:"error,omitempty"`

	// Sampling
	Sampled bool `json:"sampled"`

	// Context propagation
	BaggageItems map[string]string `json:"baggageItems,omitempty"`
}

// Span represents a span within a trace
type Span struct {
	// Span identification
	SpanID   string `json:"spanID"`
	TraceID  string `json:"traceID"`
	ParentID string `json:"parentID,omitempty"`

	// Span details
	OperationName string            `json:"operationName"`
	ComponentName string            `json:"componentName"`
	Tags          map[string]string `json:"tags"`
	Logs          []LogEntry        `json:"logs,omitempty"`

	// Timing
	StartTime time.Time     `json:"startTime"`
	Duration  time.Duration `json:"duration"`

	// Status
	Status SpanStatus `json:"status"`
	Error  string     `json:"error,omitempty"`

	// References
	References []SpanReference `json:"references,omitempty"`
}

// LogEntry represents a log entry within a span
type LogEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Fields    map[string]string `json:"fields"`
}

// SpanReference represents a reference between spans
type SpanReference struct {
	Type    string `json:"type"` // "child_of", "follows_from"
	TraceID string `json:"traceID"`
	SpanID  string `json:"spanID"`
}

// TraceStatus represents trace status
type TraceStatus string

const (
	TraceStatusActive    TraceStatus = "active"
	TraceStatusCompleted TraceStatus = "completed"
	TraceStatusError     TraceStatus = "error"
	TraceStatusTimeout   TraceStatus = "timeout"
)

// SpanStatus represents span status
type SpanStatus string

const (
	SpanStatusActive    SpanStatus = "active"
	SpanStatusCompleted SpanStatus = "completed"
	SpanStatusError     SpanStatus = "error"
)

// TraceExporter interface for exporting traces
type TraceExporter interface {
	Export(traces []Trace) error
	Close() error
}

// NewTracingManager creates a new tracing manager
func NewTracingManager(config TracingConfig) *TracingManager {
	// Set defaults
	if config.SamplingRate == 0 {
		config.SamplingRate = 0.1 // 10% default sampling
	}
	if config.BatchTimeout == 0 {
		config.BatchTimeout = 5 * time.Second
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.MaxTraces == 0 {
		config.MaxTraces = 10000
	}

	tm := &TracingManager{
		traces: make(map[string]*Trace),
		samplingConfig: SamplingConfig{
			DefaultRate:      config.SamplingRate,
			OperationRates:   make(map[string]float64),
			RateLimit:        1000,
			AdaptiveSampling: true,
		},
		exporters: make([]TraceExporter, 0),
		config:    config,
	}

	// Initialize default exporters
	tm.exporters = append(tm.exporters, NewJaegerExporter())
	tm.exporters = append(tm.exporters, NewPrometheusExporter())

	return tm
}

// StartTrace starts a new trace
func (tm *TracingManager) StartTrace(ctx context.Context, operationName string, tags map[string]string) (*Trace, context.Context) {
	traceID := tm.generateTraceID()

	// Check sampling decision
	sampled := tm.shouldSample(operationName)

	trace := &Trace{
		TraceID:       traceID,
		ServiceName:   tm.config.ServiceName,
		OperationName: operationName,
		Tags:          tags,
		StartTime:     time.Now(),
		Spans:         make([]Span, 0),
		Status:        TraceStatusActive,
		Sampled:       sampled,
		BaggageItems:  make(map[string]string),
	}

	// Store trace if sampled
	if sampled {
		tm.traceMutex.Lock()
		tm.traces[traceID] = trace
		tm.traceMutex.Unlock()
	}

	// Add trace to context
	ctx = tm.contextWithTrace(ctx, trace)

	klog.V(4).Infof("Started trace %s for operation %s (sampled: %v)", traceID, operationName, sampled)
	return trace, ctx
}

// StartSpan starts a new span within a trace
func (tm *TracingManager) StartSpan(ctx context.Context, operationName string, tags map[string]string) (*Span, context.Context) {
	trace := tm.traceFromContext(ctx)
	if trace == nil || !trace.Sampled {
		return nil, ctx
	}

	spanID := tm.generateSpanID()

	span := &Span{
		SpanID:        spanID,
		TraceID:       trace.TraceID,
		OperationName: operationName,
		ComponentName: tm.config.ServiceName,
		Tags:          tags,
		Logs:          make([]LogEntry, 0),
		StartTime:     time.Now(),
		Status:        SpanStatusActive,
		References:    make([]SpanReference, 0),
	}

	// Add span to trace
	tm.traceMutex.Lock()
	trace.Spans = append(trace.Spans, *span)
	tm.traceMutex.Unlock()

	// Add span to context
	ctx = tm.contextWithSpan(ctx, span)

	klog.V(4).Infof("Started span %s in trace %s", spanID, trace.TraceID)
	return span, ctx
}

// FinishSpan finishes a span
func (tm *TracingManager) FinishSpan(ctx context.Context, span *Span, err error) {
	if span == nil {
		return
	}

	span.Duration = time.Since(span.StartTime)

	if err != nil {
		span.Status = SpanStatusError
		span.Error = err.Error()
		span.Tags["error"] = "true"
	} else {
		span.Status = SpanStatusCompleted
	}

	klog.V(4).Infof("Finished span %s (duration: %v)", span.SpanID, span.Duration)
}

// FinishTrace finishes a trace
func (tm *TracingManager) FinishTrace(ctx context.Context, trace *Trace, err error) {
	if trace == nil || !trace.Sampled {
		return
	}

	trace.Duration = time.Since(trace.StartTime)

	if err != nil {
		trace.Status = TraceStatusError
		trace.Error = err.Error()
	} else {
		trace.Status = TraceStatusCompleted
	}

	// Export trace
	go tm.exportTrace(trace)

	// Remove from active traces
	tm.traceMutex.Lock()
	delete(tm.traces, trace.TraceID)
	tm.traceMutex.Unlock()

	klog.V(4).Infof("Finished trace %s (duration: %v)", trace.TraceID, trace.Duration)
}

// LogToSpan adds a log entry to a span
func (tm *TracingManager) LogToSpan(ctx context.Context, fields map[string]string) {
	span := tm.spanFromContext(ctx)
	if span == nil {
		return
	}

	logEntry := LogEntry{
		Timestamp: time.Now(),
		Fields:    fields,
	}

	span.Logs = append(span.Logs, logEntry)
}

// SetTag sets a tag on a span
func (tm *TracingManager) SetTag(ctx context.Context, key, value string) {
	span := tm.spanFromContext(ctx)
	if span == nil {
		return
	}

	span.Tags[key] = value
}

// SetBaggage sets baggage on a trace
func (tm *TracingManager) SetBaggage(ctx context.Context, key, value string) {
	trace := tm.traceFromContext(ctx)
	if trace == nil {
		return
	}

	trace.BaggageItems[key] = value
}

// GetBaggage gets baggage from a trace
func (tm *TracingManager) GetBaggage(ctx context.Context, key string) string {
	trace := tm.traceFromContext(ctx)
	if trace == nil {
		return ""
	}

	return trace.BaggageItems[key]
}

// GetActiveTraces returns current active traces
func (tm *TracingManager) GetActiveTraces() []Trace {
	tm.traceMutex.RLock()
	defer tm.traceMutex.RUnlock()

	traces := make([]Trace, 0, len(tm.traces))
	for _, trace := range tm.traces {
		traceCopy := *trace
		traces = append(traces, traceCopy)
	}

	return traces
}

// GetTracingMetrics returns tracing metrics
func (tm *TracingManager) GetTracingMetrics() map[string]interface{} {
	tm.traceMutex.RLock()
	defer tm.traceMutex.RUnlock()

	return map[string]interface{}{
		"active_traces":   len(tm.traces),
		"sampling_rate":   tm.config.SamplingRate,
		"service_name":    tm.config.ServiceName,
		"service_version": tm.config.ServiceVersion,
	}
}

// Private methods

func (tm *TracingManager) shouldSample(operationName string) bool {
	// Check operation-specific rate
	if rate, exists := tm.samplingConfig.OperationRates[operationName]; exists {
		return tm.sampleWithRate(rate)
	}

	// Use default rate
	return tm.sampleWithRate(tm.samplingConfig.DefaultRate)
}

func (tm *TracingManager) sampleWithRate(rate float64) bool {
	// Simple random sampling
	// In production, would use more sophisticated algorithms
	return float64(time.Now().UnixNano()%100)/100.0 < rate
}

func (tm *TracingManager) exportTrace(trace *Trace) {
	for _, exporter := range tm.exporters {
		if err := exporter.Export([]Trace{*trace}); err != nil {
			klog.ErrorS(err, "Failed to export trace", "traceID", trace.TraceID)
		}
	}
}

func (tm *TracingManager) generateTraceID() string {
	return fmt.Sprintf("trace-%d", time.Now().UnixNano())
}

func (tm *TracingManager) generateSpanID() string {
	return fmt.Sprintf("span-%d", time.Now().UnixNano())
}

// Context helpers

type contextKey string

const (
	traceContextKey contextKey = "trace"
	spanContextKey  contextKey = "span"
)

func (tm *TracingManager) contextWithTrace(ctx context.Context, trace *Trace) context.Context {
	return context.WithValue(ctx, traceContextKey, trace)
}

func (tm *TracingManager) traceFromContext(ctx context.Context) *Trace {
	if trace, ok := ctx.Value(traceContextKey).(*Trace); ok {
		return trace
	}
	return nil
}

func (tm *TracingManager) contextWithSpan(ctx context.Context, span *Span) context.Context {
	return context.WithValue(ctx, spanContextKey, span)
}

func (tm *TracingManager) spanFromContext(ctx context.Context) *Span {
	if span, ok := ctx.Value(spanContextKey).(*Span); ok {
		return span
	}
	return nil
}

// Jaeger Exporter
type JaegerExporter struct{}

func NewJaegerExporter() *JaegerExporter {
	return &JaegerExporter{}
}

func (je *JaegerExporter) Export(traces []Trace) error {
	// TODO: Implement actual Jaeger export
	klog.V(4).Infof("Exporting %d traces to Jaeger", len(traces))
	return nil
}

func (je *JaegerExporter) Close() error {
	return nil
}

// Prometheus Exporter
type PrometheusExporter struct{}

func NewPrometheusExporter() *PrometheusExporter {
	return &PrometheusExporter{}
}

func (pe *PrometheusExporter) Export(traces []Trace) error {
	// TODO: Implement actual Prometheus metrics export
	klog.V(4).Infof("Exporting %d traces to Prometheus", len(traces))
	return nil
}

func (pe *PrometheusExporter) Close() error {
	return nil
}
