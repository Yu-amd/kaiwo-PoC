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

package dashboards

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// EnterpriseDashboard provides advanced dashboard management
type EnterpriseDashboard struct {
	// Dashboard definitions
	dashboards     map[string]*Dashboard
	dashboardMutex sync.RWMutex

	// Data sources
	dataSources map[string]DataSource
	sourceMutex sync.RWMutex

	// Dashboard server
	server *DashboardServer

	// Configuration
	config DashboardConfig
}

// DashboardConfig holds dashboard configuration
type DashboardConfig struct {
	// Server settings
	Port       int    `json:"port"`
	Host       string `json:"host"`
	TLSEnabled bool   `json:"tlsEnabled"`
	CertFile   string `json:"certFile,omitempty"`
	KeyFile    string `json:"keyFile,omitempty"`

	// Authentication
	AuthEnabled  bool   `json:"authEnabled"`
	AuthProvider string `json:"authProvider"`

	// Features
	EnableAlerts    bool `json:"enableAlerts"`
	EnableExporting bool `json:"enableExporting"`
	EnableSharing   bool `json:"enableSharing"`

	// Data retention
	DataRetention time.Duration `json:"dataRetention"`

	// Performance
	MaxConcurrentQueries int           `json:"maxConcurrentQueries"`
	QueryTimeout         time.Duration `json:"queryTimeout"`
}

// Dashboard represents a monitoring dashboard
type Dashboard struct {
	// Basic information
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata"`

	// Layout
	Layout DashboardLayout `json:"layout"`
	Panels []Panel         `json:"panels"`

	// Configuration
	TimeRange TimeRange  `json:"timeRange"`
	Refresh   string     `json:"refresh"`
	Variables []Variable `json:"variables,omitempty"`

	// Access control
	Permissions []Permission `json:"permissions,omitempty"`

	// Status
	Created    time.Time `json:"created"`
	Updated    time.Time `json:"updated"`
	Version    int       `json:"version"`
	IsPublic   bool      `json:"isPublic"`
	IsEditable bool      `json:"isEditable"`
}

// DashboardLayout defines dashboard layout
type DashboardLayout struct {
	Type    string `json:"type"` // "grid", "flow", "tabs"
	Columns int    `json:"columns"`
	Rows    int    `json:"rows"`
}

// Panel represents a dashboard panel
type Panel struct {
	// Basic information
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"` // "graph", "stat", "table", "heatmap", "gauge", "alert"

	// Position and size
	Position PanelPosition `json:"position"`
	Size     PanelSize     `json:"size"`

	// Data configuration
	DataSource string      `json:"dataSource"`
	Queries    []DataQuery `json:"queries"`
	Transform  []Transform `json:"transform,omitempty"`

	// Visualization
	Visualization VisualizationConfig `json:"visualization"`

	// Alerts
	Alerts []AlertRule `json:"alerts,omitempty"`

	// Options
	Options PanelOptions `json:"options"`
}

// PanelPosition defines panel position
type PanelPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// PanelSize defines panel size
type PanelSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// DataQuery represents a data query
type DataQuery struct {
	// Query identification
	RefID string `json:"refId"`

	// Query details
	Query      string `json:"query"`
	DataSource string `json:"dataSource"`
	Format     string `json:"format"`

	// Time range
	TimeRange *TimeRange `json:"timeRange,omitempty"`

	// Query options
	MaxDataPoints int               `json:"maxDataPoints,omitempty"`
	Interval      string            `json:"interval,omitempty"`
	Variables     map[string]string `json:"variables,omitempty"`
}

// Transform represents data transformation
type Transform struct {
	Type    string                 `json:"type"`
	Options map[string]interface{} `json:"options"`
}

// VisualizationConfig defines visualization settings
type VisualizationConfig struct {
	// Chart type specific settings
	ChartType string                 `json:"chartType"`
	Options   map[string]interface{} `json:"options"`

	// Axes configuration
	XAxis AxisConfig `json:"xAxis"`
	YAxis AxisConfig `json:"yAxis"`

	// Series configuration
	Series []SeriesConfig `json:"series,omitempty"`

	// Styling
	Colors []string     `json:"colors,omitempty"`
	Theme  string       `json:"theme"`
	Legend LegendConfig `json:"legend"`

	// Thresholds
	Thresholds []Threshold `json:"thresholds,omitempty"`
}

// AxisConfig defines axis configuration
type AxisConfig struct {
	Title    string  `json:"title"`
	Min      float64 `json:"min,omitempty"`
	Max      float64 `json:"max,omitempty"`
	Unit     string  `json:"unit,omitempty"`
	LogScale bool    `json:"logScale,omitempty"`
}

// SeriesConfig defines series configuration
type SeriesConfig struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Color       string  `json:"color,omitempty"`
	LineWidth   int     `json:"lineWidth,omitempty"`
	FillOpacity float64 `json:"fillOpacity,omitempty"`
}

// LegendConfig defines legend configuration
type LegendConfig struct {
	Show     bool     `json:"show"`
	Position string   `json:"position"`
	Values   []string `json:"values,omitempty"`
}

// Threshold defines visualization thresholds
type Threshold struct {
	Value float64 `json:"value"`
	Color string  `json:"color"`
	Label string  `json:"label,omitempty"`
}

// PanelOptions defines panel-specific options
type PanelOptions struct {
	ShowTitle       bool                   `json:"showTitle"`
	ShowDescription bool                   `json:"showDescription"`
	Transparent     bool                   `json:"transparent"`
	Custom          map[string]interface{} `json:"custom,omitempty"`
}

// TimeRange defines a time range
type TimeRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Variable represents a dashboard variable
type Variable struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"` // "query", "constant", "interval", "datasource"
	Label      string   `json:"label"`
	Query      string   `json:"query,omitempty"`
	Options    []string `json:"options,omitempty"`
	Multi      bool     `json:"multi"`
	IncludeAll bool     `json:"includeAll"`
	Current    string   `json:"current"`
	Default    string   `json:"default"`
}

// Permission defines dashboard access permissions
type Permission struct {
	Role       string `json:"role"`
	UserID     string `json:"userId,omitempty"`
	GroupID    string `json:"groupId,omitempty"`
	Permission string `json:"permission"` // "view", "edit", "admin"
}

// AlertRule defines alerting rules for panels
type AlertRule struct {
	ID                  string           `json:"id"`
	Name                string           `json:"name"`
	Frequency           string           `json:"frequency"`
	Conditions          []AlertCondition `json:"conditions"`
	Notifications       []Notification   `json:"notifications"`
	ExecutionErrorState string           `json:"executionErrorState"`
	NoDataState         string           `json:"noDataState"`
	For                 string           `json:"for"`
}

// AlertCondition defines alert conditions
type AlertCondition struct {
	Query     DataQuery `json:"query"`
	Reducer   Reducer   `json:"reducer"`
	Evaluator Evaluator `json:"evaluator"`
}

// Reducer defines how to reduce query results
type Reducer struct {
	Type   string                 `json:"type"` // "avg", "min", "max", "sum", "count", "last", "median"
	Params map[string]interface{} `json:"params,omitempty"`
}

// Evaluator defines how to evaluate reduced values
type Evaluator struct {
	Type   string    `json:"type"` // "gt", "lt", "eq", "ne", "within_range", "outside_range"
	Params []float64 `json:"params"`
}

// Notification defines alert notifications
type Notification struct {
	Type     string            `json:"type"` // "email", "slack", "webhook", "pagerduty"
	Settings map[string]string `json:"settings"`
}

// DataSource interface for dashboard data sources
type DataSource interface {
	Query(query DataQuery) (QueryResult, error)
	TestConnection() error
	GetMetadata() DataSourceMetadata
}

// QueryResult represents query results
type QueryResult struct {
	Series []TimeSeries `json:"series"`
	Tables []Table      `json:"tables,omitempty"`
	Error  string       `json:"error,omitempty"`
}

// TimeSeries represents time series data
type TimeSeries struct {
	Name   string            `json:"name"`
	Points []DataPoint       `json:"points"`
	Tags   map[string]string `json:"tags,omitempty"`
}

// DataPoint represents a data point
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// Table represents tabular data
type Table struct {
	Columns []Column   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

// Column represents a table column
type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// DataSourceMetadata provides metadata about a data source
type DataSourceMetadata struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

// DashboardServer provides HTTP API for dashboards
type DashboardServer struct {
	dashboard *EnterpriseDashboard
	config    DashboardConfig
}

// NewEnterpriseDashboard creates a new enterprise dashboard
func NewEnterpriseDashboard(config DashboardConfig) *EnterpriseDashboard {
	// Set defaults
	if config.Port == 0 {
		config.Port = 3000
	}
	if config.Host == "" {
		config.Host = "0.0.0.0"
	}
	if config.DataRetention == 0 {
		config.DataRetention = 30 * 24 * time.Hour // 30 days
	}
	if config.MaxConcurrentQueries == 0 {
		config.MaxConcurrentQueries = 100
	}
	if config.QueryTimeout == 0 {
		config.QueryTimeout = 30 * time.Second
	}

	ed := &EnterpriseDashboard{
		dashboards:  make(map[string]*Dashboard),
		dataSources: make(map[string]DataSource),
		config:      config,
	}

	// Initialize server
	ed.server = &DashboardServer{
		dashboard: ed,
		config:    config,
	}

	// Create default dashboards
	ed.createDefaultDashboards()

	return ed
}

// CreateDashboard creates a new dashboard
func (ed *EnterpriseDashboard) CreateDashboard(dashboard *Dashboard) error {
	ed.dashboardMutex.Lock()
	defer ed.dashboardMutex.Unlock()

	if dashboard.ID == "" {
		dashboard.ID = ed.generateDashboardID()
	}

	// Set timestamps
	now := time.Now()
	dashboard.Created = now
	dashboard.Updated = now
	dashboard.Version = 1

	// Validate dashboard
	if err := ed.validateDashboard(dashboard); err != nil {
		return fmt.Errorf("invalid dashboard: %v", err)
	}

	ed.dashboards[dashboard.ID] = dashboard

	klog.Infof("Created dashboard %s (%s)", dashboard.ID, dashboard.Name)
	return nil
}

// GetDashboard retrieves a dashboard by ID
func (ed *EnterpriseDashboard) GetDashboard(id string) (*Dashboard, error) {
	ed.dashboardMutex.RLock()
	defer ed.dashboardMutex.RUnlock()

	dashboard, exists := ed.dashboards[id]
	if !exists {
		return nil, fmt.Errorf("dashboard %s not found", id)
	}

	// Return a copy
	dashboardCopy := *dashboard
	return &dashboardCopy, nil
}

// UpdateDashboard updates an existing dashboard
func (ed *EnterpriseDashboard) UpdateDashboard(dashboard *Dashboard) error {
	ed.dashboardMutex.Lock()
	defer ed.dashboardMutex.Unlock()

	existing, exists := ed.dashboards[dashboard.ID]
	if !exists {
		return fmt.Errorf("dashboard %s not found", dashboard.ID)
	}

	// Preserve creation info
	dashboard.Created = existing.Created
	dashboard.Updated = time.Now()
	dashboard.Version = existing.Version + 1

	// Validate dashboard
	if err := ed.validateDashboard(dashboard); err != nil {
		return fmt.Errorf("invalid dashboard: %v", err)
	}

	ed.dashboards[dashboard.ID] = dashboard

	klog.Infof("Updated dashboard %s (%s)", dashboard.ID, dashboard.Name)
	return nil
}

// DeleteDashboard deletes a dashboard
func (ed *EnterpriseDashboard) DeleteDashboard(id string) error {
	ed.dashboardMutex.Lock()
	defer ed.dashboardMutex.Unlock()

	if _, exists := ed.dashboards[id]; !exists {
		return fmt.Errorf("dashboard %s not found", id)
	}

	delete(ed.dashboards, id)

	klog.Infof("Deleted dashboard %s", id)
	return nil
}

// ListDashboards returns all dashboards
func (ed *EnterpriseDashboard) ListDashboards() []*Dashboard {
	ed.dashboardMutex.RLock()
	defer ed.dashboardMutex.RUnlock()

	dashboards := make([]*Dashboard, 0, len(ed.dashboards))
	for _, dashboard := range ed.dashboards {
		dashboardCopy := *dashboard
		dashboards = append(dashboards, &dashboardCopy)
	}

	return dashboards
}

// RegisterDataSource registers a new data source
func (ed *EnterpriseDashboard) RegisterDataSource(name string, dataSource DataSource) error {
	ed.sourceMutex.Lock()
	defer ed.sourceMutex.Unlock()

	// Test connection
	if err := dataSource.TestConnection(); err != nil {
		return fmt.Errorf("data source connection test failed: %v", err)
	}

	ed.dataSources[name] = dataSource

	klog.Infof("Registered data source %s", name)
	return nil
}

// QueryData queries data from a data source
func (ed *EnterpriseDashboard) QueryData(query DataQuery) (QueryResult, error) {
	ed.sourceMutex.RLock()
	dataSource, exists := ed.dataSources[query.DataSource]
	ed.sourceMutex.RUnlock()

	if !exists {
		return QueryResult{}, fmt.Errorf("data source %s not found", query.DataSource)
	}

	return dataSource.Query(query)
}

// ExportDashboard exports a dashboard to JSON
func (ed *EnterpriseDashboard) ExportDashboard(id string) (string, error) {
	dashboard, err := ed.GetDashboard(id)
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(dashboard, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal dashboard: %v", err)
	}

	return string(data), nil
}

// ImportDashboard imports a dashboard from JSON
func (ed *EnterpriseDashboard) ImportDashboard(data string) error {
	var dashboard Dashboard
	if err := json.Unmarshal([]byte(data), &dashboard); err != nil {
		return fmt.Errorf("failed to unmarshal dashboard: %v", err)
	}

	// Generate new ID to avoid conflicts
	dashboard.ID = ed.generateDashboardID()

	return ed.CreateDashboard(&dashboard)
}

// Start starts the dashboard server
func (ed *EnterpriseDashboard) Start() error {
	// TODO: Implement HTTP server
	klog.Infof("Starting dashboard server on %s:%d", ed.config.Host, ed.config.Port)
	return nil
}

// Stop stops the dashboard server
func (ed *EnterpriseDashboard) Stop() error {
	// TODO: Implement server shutdown
	klog.Info("Stopping dashboard server")
	return nil
}

// Private methods

func (ed *EnterpriseDashboard) createDefaultDashboards() {
	// Create Kaiwo Overview Dashboard
	overview := &Dashboard{
		ID:          "kaiwo-overview",
		Name:        "Kaiwo Overview",
		Description: "High-level overview of Kaiwo scheduler performance and health",
		Tags:        []string{"kaiwo", "overview", "scheduler"},
		Layout: DashboardLayout{
			Type:    "grid",
			Columns: 12,
			Rows:    8,
		},
		TimeRange: TimeRange{
			From: "now-1h",
			To:   "now",
		},
		Refresh: "30s",
		Panels: []Panel{
			{
				ID:         "scheduler-status",
				Title:      "Scheduler Status",
				Type:       "stat",
				Position:   PanelPosition{X: 0, Y: 0},
				Size:       PanelSize{Width: 3, Height: 2},
				DataSource: "prometheus",
				Queries: []DataQuery{
					{
						RefID:      "A",
						Query:      "kaiwo_scheduler_status",
						DataSource: "prometheus",
					},
				},
			},
			{
				ID:         "job-scheduling-rate",
				Title:      "Job Scheduling Rate",
				Type:       "graph",
				Position:   PanelPosition{X: 3, Y: 0},
				Size:       PanelSize{Width: 6, Height: 3},
				DataSource: "prometheus",
				Queries: []DataQuery{
					{
						RefID:      "A",
						Query:      "rate(kaiwo_jobs_scheduled_total[5m])",
						DataSource: "prometheus",
					},
				},
			},
			{
				ID:         "gpu-utilization",
				Title:      "GPU Utilization",
				Type:       "gauge",
				Position:   PanelPosition{X: 9, Y: 0},
				Size:       PanelSize{Width: 3, Height: 3},
				DataSource: "prometheus",
				Queries: []DataQuery{
					{
						RefID:      "A",
						Query:      "avg(kaiwo_gpu_utilization_percent)",
						DataSource: "prometheus",
					},
				},
			},
		},
	}

	ed.CreateDashboard(overview)

	// Create ML Analytics Dashboard
	mlAnalytics := &Dashboard{
		ID:          "ml-analytics",
		Name:        "ML Analytics",
		Description: "Machine learning prediction and analytics performance",
		Tags:        []string{"kaiwo", "ml", "analytics", "predictions"},
		Layout: DashboardLayout{
			Type:    "grid",
			Columns: 12,
			Rows:    6,
		},
		TimeRange: TimeRange{
			From: "now-6h",
			To:   "now",
		},
		Refresh: "1m",
		Panels: []Panel{
			{
				ID:         "prediction-accuracy",
				Title:      "Prediction Accuracy",
				Type:       "graph",
				Position:   PanelPosition{X: 0, Y: 0},
				Size:       PanelSize{Width: 6, Height: 3},
				DataSource: "prometheus",
				Queries: []DataQuery{
					{
						RefID:      "A",
						Query:      "kaiwo_ml_prediction_accuracy",
						DataSource: "prometheus",
					},
				},
			},
			{
				ID:         "anomaly-detection",
				Title:      "Anomaly Detection",
				Type:       "heatmap",
				Position:   PanelPosition{X: 6, Y: 0},
				Size:       PanelSize{Width: 6, Height: 3},
				DataSource: "prometheus",
				Queries: []DataQuery{
					{
						RefID:      "A",
						Query:      "kaiwo_anomaly_score",
						DataSource: "prometheus",
					},
				},
			},
		},
	}

	ed.CreateDashboard(mlAnalytics)

	// Create Federation Dashboard
	federation := &Dashboard{
		ID:          "federation",
		Name:        "Multi-Cluster Federation",
		Description: "Cross-cluster workload management and federation status",
		Tags:        []string{"kaiwo", "federation", "multi-cluster"},
		Layout: DashboardLayout{
			Type:    "grid",
			Columns: 12,
			Rows:    6,
		},
		TimeRange: TimeRange{
			From: "now-2h",
			To:   "now",
		},
		Refresh: "1m",
		Panels: []Panel{
			{
				ID:         "cluster-health",
				Title:      "Cluster Health Status",
				Type:       "table",
				Position:   PanelPosition{X: 0, Y: 0},
				Size:       PanelSize{Width: 6, Height: 4},
				DataSource: "prometheus",
				Queries: []DataQuery{
					{
						RefID:      "A",
						Query:      "kaiwo_cluster_health_status",
						DataSource: "prometheus",
					},
				},
			},
			{
				ID:         "cross-cluster-workloads",
				Title:      "Cross-Cluster Workloads",
				Type:       "graph",
				Position:   PanelPosition{X: 6, Y: 0},
				Size:       PanelSize{Width: 6, Height: 4},
				DataSource: "prometheus",
				Queries: []DataQuery{
					{
						RefID:      "A",
						Query:      "kaiwo_federated_workloads_total",
						DataSource: "prometheus",
					},
				},
			},
		},
	}

	ed.CreateDashboard(federation)
}

func (ed *EnterpriseDashboard) validateDashboard(dashboard *Dashboard) error {
	if dashboard.Name == "" {
		return fmt.Errorf("dashboard name cannot be empty")
	}

	if len(dashboard.Panels) == 0 {
		return fmt.Errorf("dashboard must have at least one panel")
	}

	// Validate panels
	for _, panel := range dashboard.Panels {
		if panel.Title == "" {
			return fmt.Errorf("panel title cannot be empty")
		}

		if len(panel.Queries) == 0 {
			return fmt.Errorf("panel must have at least one query")
		}

		// Validate data source exists
		for _, query := range panel.Queries {
			ed.sourceMutex.RLock()
			_, exists := ed.dataSources[query.DataSource]
			ed.sourceMutex.RUnlock()

			if !exists {
				return fmt.Errorf("data source %s not found", query.DataSource)
			}
		}
	}

	return nil
}

func (ed *EnterpriseDashboard) generateDashboardID() string {
	return fmt.Sprintf("dashboard-%d", time.Now().UnixNano())
}
