package config

import (
	"time"
)

// Config is the root configuration structure.
type Config struct {
	Name           string                    `yaml:"name"`
	Version        string                    `yaml:"version"`
	Discovery      DiscoveryConfig           `yaml:"discovery"`
	Infrastructure map[string]InfraConfig    `yaml:"infrastructure"`
	Scenarios      []ScenarioConfig          `yaml:"scenarios"`
	ChaosProfiles  map[string]ChaosProfile   `yaml:"chaos_profiles"`
	MockProfiles   map[string]MockProfile    `yaml:"mock_profiles"`
	Flags          FlagsConfig               `yaml:"flags"`
	Options        map[string]OptionConfig   `yaml:"options"`
	Bundles        BundlesConfig             `yaml:"bundles"`
	Secrets        SecretsConfig             `yaml:"secrets"`
	Execution      ExecutionConfig           `yaml:"execution"`
	Results        ResultsConfig             `yaml:"results"`
	Notifications  NotificationsConfig       `yaml:"notifications"`
}

// DiscoveryConfig configures component discovery.
type DiscoveryConfig struct {
	Paths   []string `yaml:"paths"`
	Exclude []string `yaml:"exclude"`
}

// InfraConfig configures an infrastructure resource.
type InfraConfig struct {
	Provider    string            `yaml:"provider"`
	Image       string            `yaml:"image,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	Ports       []PortConfig      `yaml:"ports,omitempty"`
	Volumes     []VolumeConfig    `yaml:"volumes,omitempty"`
	HealthCheck HealthCheckConfig `yaml:"health_check,omitempty"`
	Reuse       ReuseConfig       `yaml:"reuse,omitempty"`
	DependsOn   []string          `yaml:"depends_on,omitempty"`
	Resources   ResourcesConfig   `yaml:"resources,omitempty"`
}

// PortConfig configures a port mapping.
type PortConfig struct {
	Container int    `yaml:"container"`
	Host      int    `yaml:"host,omitempty"`
	Protocol  string `yaml:"protocol,omitempty"`
}

// VolumeConfig configures a volume mount.
type VolumeConfig struct {
	Source   string `yaml:"source"`
	Target   string `yaml:"target"`
	ReadOnly bool   `yaml:"read_only,omitempty"`
}

// HealthCheckConfig configures health checking.
type HealthCheckConfig struct {
	Command  []string      `yaml:"command,omitempty"`
	Endpoint string        `yaml:"endpoint,omitempty"`
	Interval time.Duration `yaml:"interval,omitempty"`
	Timeout  time.Duration `yaml:"timeout,omitempty"`
	Retries  int           `yaml:"retries,omitempty"`
}

// ReuseConfig configures resource reuse.
type ReuseConfig struct {
	Enabled bool          `yaml:"enabled"`
	TTL     time.Duration `yaml:"ttl,omitempty"`
	Key     string        `yaml:"key,omitempty"`
}

// ResourcesConfig configures resource limits.
type ResourcesConfig struct {
	Memory string `yaml:"memory,omitempty"`
	CPU    string `yaml:"cpu,omitempty"`
}

// ScenarioConfig configures a test scenario.
type ScenarioConfig struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description,omitempty"`
	Timeout     time.Duration `yaml:"timeout,omitempty"`
	Tags        []string      `yaml:"tags,omitempty"`

	// Flow definition
	Flow         []FlowItemConfig `yaml:"flow"`
	TeardownFlow []FlowItemConfig `yaml:"teardown,omitempty"`

	// Execution modifiers
	Flags         map[string]any `yaml:"flags,omitempty"`
	Options       []string       `yaml:"options,omitempty"`
	ChaosProfiles []string       `yaml:"chaos_profiles,omitempty"`
	MockProfiles  []string       `yaml:"mock_profiles,omitempty"`

	// Conditions
	SkipIf     []ConditionConfig `yaml:"skip_if,omitempty"`
	SkipUnless []ConditionConfig `yaml:"skip_unless,omitempty"`

	// Matrix for parameterized tests
	Matrix map[string][]any `yaml:"matrix,omitempty"`

	// Inheritance
	Extends  string `yaml:"extends,omitempty"`
	Abstract bool   `yaml:"abstract,omitempty"`
}

// FlowItemConfig configures a single flow item.
type FlowItemConfig struct {
	// Component type (only one should be set)
	Setup      string `yaml:"setup,omitempty"`
	Task       string `yaml:"task,omitempty"`
	Validation string `yaml:"validation,omitempty"`
	Step       string `yaml:"step,omitempty"`
	Rollup     string `yaml:"rollup,omitempty"`
	Teardown   string `yaml:"teardown,omitempty"`

	// Common properties
	Timeout   time.Duration  `yaml:"timeout,omitempty"`
	DependsOn []string       `yaml:"depends_on,omitempty"`
	Params    map[string]any `yaml:"params,omitempty"`
	Parallel  bool           `yaml:"parallel,omitempty"`
}

// ConditionConfig configures a skip condition.
type ConditionConfig struct {
	Expression string `yaml:"expression,omitempty"`
	Env        string `yaml:"env,omitempty"`
	Flag       string `yaml:"flag,omitempty"`
	Reason     string `yaml:"reason,omitempty"`
}

// ChaosProfile configures chaos engineering settings.
type ChaosProfile struct {
	Name        string              `yaml:"name,omitempty"`
	Description string              `yaml:"description,omitempty"`
	Network     NetworkChaosConfig  `yaml:"network,omitempty"`
	Resource    ResourceChaosConfig `yaml:"resource,omitempty"`
	Custom      CustomChaosConfig   `yaml:"custom,omitempty"`
}

// NetworkChaosConfig configures network chaos.
type NetworkChaosConfig struct {
	Latency    LatencyConfig    `yaml:"latency,omitempty"`
	PacketLoss PacketLossConfig `yaml:"packet_loss,omitempty"`
	Partition  PartitionConfig  `yaml:"partition,omitempty"`
}

// LatencyConfig configures latency injection.
type LatencyConfig struct {
	Enabled bool          `yaml:"enabled"`
	Min     time.Duration `yaml:"min"`
	Max     time.Duration `yaml:"max"`
	Jitter  float64       `yaml:"jitter,omitempty"`
}

// PacketLossConfig configures packet loss.
type PacketLossConfig struct {
	Enabled    bool    `yaml:"enabled"`
	Percentage float64 `yaml:"percentage"`
}

// PartitionConfig configures network partitions.
type PartitionConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Duration time.Duration `yaml:"duration"`
	Targets  []string      `yaml:"targets,omitempty"`
}

// ResourceChaosConfig configures resource chaos.
type ResourceChaosConfig struct {
	CPU    CPUChaosConfig    `yaml:"cpu,omitempty"`
	Memory MemoryChaosConfig `yaml:"memory,omitempty"`
	IO     IOChaosConfig     `yaml:"io,omitempty"`
}

// CPUChaosConfig configures CPU chaos.
type CPUChaosConfig struct {
	Enabled    bool          `yaml:"enabled"`
	Percentage int           `yaml:"percentage"`
	Duration   time.Duration `yaml:"duration,omitempty"`
}

// MemoryChaosConfig configures memory chaos.
type MemoryChaosConfig struct {
	Enabled    bool          `yaml:"enabled"`
	Percentage int           `yaml:"percentage"`
	Duration   time.Duration `yaml:"duration,omitempty"`
}

// IOChaosConfig configures I/O chaos.
type IOChaosConfig struct {
	Enabled    bool          `yaml:"enabled"`
	Percentage int           `yaml:"percentage"`
	Duration   time.Duration `yaml:"duration,omitempty"`
}

// CustomChaosConfig configures custom chaos operations.
type CustomChaosConfig struct {
	Before []CustomHookConfig `yaml:"before,omitempty"`
	During []CustomHookConfig `yaml:"during,omitempty"`
	After  []CustomHookConfig `yaml:"after,omitempty"`
}

// CustomHookConfig configures a custom chaos hook.
type CustomHookConfig struct {
	Name    string            `yaml:"name"`
	Command string            `yaml:"command,omitempty"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

// MockProfile configures mocking behavior.
type MockProfile struct {
	Name        string       `yaml:"name,omitempty"`
	Description string       `yaml:"description,omitempty"`
	Services    []MockConfig `yaml:"services"`
}

// MockConfig configures a mock service.
type MockConfig struct {
	Name        string             `yaml:"name"`
	Type        string             `yaml:"type"`
	Rules       []MockRuleConfig   `yaml:"rules,omitempty"`
	Fallback    MockFallbackConfig `yaml:"fallback,omitempty"`
	Passthrough bool               `yaml:"passthrough,omitempty"`
}

// MockRuleConfig configures a mock rule.
type MockRuleConfig struct {
	Match    MockMatchConfig    `yaml:"match"`
	Response MockResponseConfig `yaml:"response"`
	Delay    time.Duration      `yaml:"delay,omitempty"`
}

// MockMatchConfig configures mock matching.
type MockMatchConfig struct {
	Method   string            `yaml:"method,omitempty"`
	Path     string            `yaml:"path,omitempty"`
	Headers  map[string]string `yaml:"headers,omitempty"`
	Body     string            `yaml:"body,omitempty"`
	BodyJSON map[string]any    `yaml:"body_json,omitempty"`
}

// MockResponseConfig configures mock response.
type MockResponseConfig struct {
	Status  int               `yaml:"status"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Body    string            `yaml:"body,omitempty"`
	File    string            `yaml:"file,omitempty"`
}

// MockFallbackConfig configures mock fallback behavior.
type MockFallbackConfig struct {
	Action string `yaml:"action"` // "error", "passthrough", "default"
	Status int    `yaml:"status,omitempty"`
	Body   string `yaml:"body,omitempty"`
}

// GetComponentName returns the component name from the flow item.
func (f FlowItemConfig) GetComponentName() string {
	if f.Setup != "" {
		return f.Setup
	}
	if f.Task != "" {
		return f.Task
	}
	if f.Validation != "" {
		return f.Validation
	}
	if f.Step != "" {
		return f.Step
	}
	if f.Rollup != "" {
		return f.Rollup
	}
	if f.Teardown != "" {
		return f.Teardown
	}
	return ""
}

// GetComponentType returns the type of the flow item.
func (f FlowItemConfig) GetComponentType() string {
	if f.Setup != "" {
		return "setup"
	}
	if f.Task != "" {
		return "task"
	}
	if f.Validation != "" {
		return "validation"
	}
	if f.Step != "" {
		return "step"
	}
	if f.Rollup != "" {
		return "rollup"
	}
	if f.Teardown != "" {
		return "teardown"
	}
	return ""
}

// FlagsConfig configures test flags.
type FlagsConfig struct {
	Definitions map[string]FlagDefinition `yaml:"definitions,omitempty"`
	Defaults    map[string]any            `yaml:"defaults,omitempty"`
}

// FlagDefinition defines a flag.
type FlagDefinition struct {
	Type        string   `yaml:"type"` // bool, string, int, float, []string
	Default     any      `yaml:"default,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Required    bool     `yaml:"required,omitempty"`
	Choices     []string `yaml:"choices,omitempty"`
}

// OptionConfig configures a named option.
type OptionConfig struct {
	Description string         `yaml:"description,omitempty"`
	Flags       map[string]any `yaml:"flags,omitempty"`
	Middleware  []string       `yaml:"middleware,omitempty"`
	Tags        []string       `yaml:"tags,omitempty"`
}

// BundlesConfig configures reusable bundles.
type BundlesConfig struct {
	Infrastructure map[string][]string `yaml:"infrastructure,omitempty"`
	Flags          map[string][]string `yaml:"flags,omitempty"`
	Options        map[string][]string `yaml:"options,omitempty"`
	Middleware     map[string][]string `yaml:"middleware,omitempty"`
}

// SecretsConfig configures secret management.
type SecretsConfig struct {
	Provider      string            `yaml:"provider"`
	Path          string            `yaml:"path,omitempty"`
	Mapping       map[string]string `yaml:"mapping,omitempty"`
	FallbackToEnv bool              `yaml:"fallback_to_env,omitempty"`
	Vault         *VaultConfig      `yaml:"vault,omitempty"`
}

// VaultConfig configures HashiCorp Vault.
type VaultConfig struct {
	Address string `yaml:"address"`
	Token   string `yaml:"token,omitempty"`
	Path    string `yaml:"path,omitempty"`
}

// ExecutionConfig configures test execution.
type ExecutionConfig struct {
	Parallelism      int           `yaml:"parallelism,omitempty"`
	DefaultTimeout   time.Duration `yaml:"default_timeout,omitempty"`
	RetryConfig      RetryConfig   `yaml:"retry,omitempty"`
	TeardownMode     string        `yaml:"teardown_mode,omitempty"` // "always", "on_failure", "never"
	FailFast         bool          `yaml:"fail_fast,omitempty"`
	ShuffleScenarios bool          `yaml:"shuffle_scenarios,omitempty"`
}

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxRetries int           `yaml:"max_retries,omitempty"`
	Backoff    BackoffConfig `yaml:"backoff,omitempty"`
}

// BackoffConfig configures backoff behavior.
type BackoffConfig struct {
	Type       string        `yaml:"type"` // "constant", "exponential", "linear"
	Initial    time.Duration `yaml:"initial,omitempty"`
	Max        time.Duration `yaml:"max,omitempty"`
	Multiplier float64       `yaml:"multiplier,omitempty"`
	Jitter     bool          `yaml:"jitter,omitempty"`
}

// ResultsConfig configures results storage.
type ResultsConfig struct {
	Storage   StorageConfig   `yaml:"storage,omitempty"`
	Reports   []ReportConfig  `yaml:"reports,omitempty"`
	Retention RetentionConfig `yaml:"retention,omitempty"`
}

// StorageConfig configures storage backend.
type StorageConfig struct {
	Type   string            `yaml:"type"` // "file", "s3", "gcs", "database"
	Path   string            `yaml:"path,omitempty"`
	Bucket string            `yaml:"bucket,omitempty"`
	Region string            `yaml:"region,omitempty"`
	Config map[string]string `yaml:"config,omitempty"`
}

// ReportConfig configures report generation.
type ReportConfig struct {
	Format string `yaml:"format"` // "junit", "json", "html", "markdown"
	Path   string `yaml:"path,omitempty"`
}

// RetentionConfig configures result retention.
type RetentionConfig struct {
	Days    int  `yaml:"days,omitempty"`
	Count   int  `yaml:"count,omitempty"`
	Cleanup bool `yaml:"cleanup,omitempty"`
}

// NotificationsConfig configures notifications.
type NotificationsConfig struct {
	Slack   SlackNotificationConfig   `yaml:"slack,omitempty"`
	Email   EmailNotificationConfig   `yaml:"email,omitempty"`
	Webhook WebhookNotificationConfig `yaml:"webhook,omitempty"`
}

// SlackNotificationConfig configures Slack notifications.
type SlackNotificationConfig struct {
	Enabled    bool     `yaml:"enabled"`
	WebhookURL string   `yaml:"webhook_url,omitempty"`
	Channel    string   `yaml:"channel,omitempty"`
	OnEvents   []string `yaml:"on_events,omitempty"` // "start", "complete", "failure"
}

// EmailNotificationConfig configures email notifications.
type EmailNotificationConfig struct {
	Enabled    bool     `yaml:"enabled"`
	SMTPServer string   `yaml:"smtp_server,omitempty"`
	From       string   `yaml:"from,omitempty"`
	To         []string `yaml:"to,omitempty"`
	OnEvents   []string `yaml:"on_events,omitempty"`
}

// WebhookNotificationConfig configures webhook notifications.
type WebhookNotificationConfig struct {
	Enabled  bool              `yaml:"enabled"`
	URL      string            `yaml:"url,omitempty"`
	Headers  map[string]string `yaml:"headers,omitempty"`
	OnEvents []string          `yaml:"on_events,omitempty"`
}
