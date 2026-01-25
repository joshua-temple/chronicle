package config

import (
	"time"
)

// Config is the root configuration structure.
type Config struct {
	Name           string                    `yaml:"name" json:"name"`
	Version        string                    `yaml:"version" json:"version"`
	Discovery      DiscoveryConfig           `yaml:"discovery" json:"discovery"`
	Infrastructure map[string]InfraConfig    `yaml:"infrastructure" json:"infrastructure"`
	Scenarios      []ScenarioConfig          `yaml:"scenarios" json:"scenarios"`
	ChaosProfiles  map[string]ChaosProfile   `yaml:"chaos_profiles" json:"chaos_profiles"`
	MockProfiles   map[string]MockProfile    `yaml:"mock_profiles" json:"mock_profiles"`
	Flags          FlagsConfig               `yaml:"flags" json:"flags"`
	Options        map[string]OptionConfig   `yaml:"options" json:"options"`
	Bundles        BundlesConfig             `yaml:"bundles" json:"bundles"`
	Secrets        SecretsConfig             `yaml:"secrets" json:"secrets"`
	Execution      ExecutionConfig           `yaml:"execution" json:"execution"`
	Results        ResultsConfig             `yaml:"results" json:"results"`
	Notifications  NotificationsConfig       `yaml:"notifications" json:"notifications"`
}

// DiscoveryConfig configures component discovery.
type DiscoveryConfig struct {
	Paths   []string `yaml:"paths" json:"paths"`
	Exclude []string `yaml:"exclude" json:"exclude"`
}

// InfraConfig configures an infrastructure resource.
type InfraConfig struct {
	Provider    string            `yaml:"provider" json:"provider"`
	Image       string            `yaml:"image,omitempty" json:"image,omitempty"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Ports       []PortConfig      `yaml:"ports,omitempty" json:"ports,omitempty"`
	Volumes     []VolumeConfig    `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	HealthCheck HealthCheckConfig `yaml:"health_check,omitempty" json:"health_check,omitempty"`
	Reuse       ReuseConfig       `yaml:"reuse,omitempty" json:"reuse,omitempty"`
	DependsOn   []string          `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Resources   ResourcesConfig   `yaml:"resources,omitempty" json:"resources,omitempty"`

	// ComposeFile references an existing docker-compose.yml file.
	// When set, Chronicle uses TestContainers' compose support instead of individual containers.
	ComposeFile string `yaml:"compose_file,omitempty" json:"compose_file,omitempty"`

	// Services filters which services to start from the compose file.
	// If empty, all services are started.
	Services []string `yaml:"services,omitempty" json:"services,omitempty"`
}

// PortConfig configures a port mapping.
type PortConfig struct {
	Container int    `yaml:"container" json:"container"`
	Host      int    `yaml:"host,omitempty" json:"host,omitempty"`
	Protocol  string `yaml:"protocol,omitempty" json:"protocol,omitempty"`
}

// VolumeConfig configures a volume mount.
type VolumeConfig struct {
	Source   string `yaml:"source" json:"source"`
	Target   string `yaml:"target" json:"target"`
	ReadOnly bool   `yaml:"read_only,omitempty" json:"read_only,omitempty"`
}

// HealthCheckConfig configures health checking.
type HealthCheckConfig struct {
	Command  []string      `yaml:"command,omitempty" json:"command,omitempty"`
	Endpoint string        `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	Interval time.Duration `yaml:"interval,omitempty" json:"interval,omitempty"`
	Timeout  time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retries  int           `yaml:"retries,omitempty" json:"retries,omitempty"`
}

// ReuseConfig configures resource reuse.
type ReuseConfig struct {
	Enabled bool          `yaml:"enabled" json:"enabled"`
	TTL     time.Duration `yaml:"ttl,omitempty" json:"ttl,omitempty"`
	Key     string        `yaml:"key,omitempty" json:"key,omitempty"`
}

// ResourcesConfig configures resource limits.
type ResourcesConfig struct {
	Memory string `yaml:"memory,omitempty" json:"memory,omitempty"`
	CPU    string `yaml:"cpu,omitempty" json:"cpu,omitempty"`
}

// ScenarioConfig configures a test scenario.
type ScenarioConfig struct {
	Name        string        `yaml:"name" json:"name"`
	Description string        `yaml:"description,omitempty" json:"description,omitempty"`
	Timeout     time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Tags        []string      `yaml:"tags,omitempty" json:"tags,omitempty"`

	// Flow definition
	Flow         []FlowItemConfig `yaml:"flow" json:"flow"`
	TeardownFlow []FlowItemConfig `yaml:"teardown,omitempty" json:"teardown,omitempty"`

	// Execution modifiers
	Flags         map[string]any `yaml:"flags,omitempty" json:"flags,omitempty"`
	Options       []string       `yaml:"options,omitempty" json:"options,omitempty"`
	ChaosProfiles []string       `yaml:"chaos_profiles,omitempty" json:"chaos_profiles,omitempty"`
	MockProfiles  []string       `yaml:"mock_profiles,omitempty" json:"mock_profiles,omitempty"`

	// Conditions
	SkipIf     []ConditionConfig `yaml:"skip_if,omitempty" json:"skip_if,omitempty"`
	SkipUnless []ConditionConfig `yaml:"skip_unless,omitempty" json:"skip_unless,omitempty"`

	// Matrix for parameterized tests
	Matrix map[string][]any `yaml:"matrix,omitempty" json:"matrix,omitempty"`

	// Inheritance
	Extends  string `yaml:"extends,omitempty" json:"extends,omitempty"`
	Abstract bool   `yaml:"abstract,omitempty" json:"abstract,omitempty"`
}

// FlowItemConfig configures a single flow item.
type FlowItemConfig struct {
	// Component type (only one should be set)
	Setup      string `yaml:"setup,omitempty" json:"setup,omitempty"`
	Task       string `yaml:"task,omitempty" json:"task,omitempty"`
	Validation string `yaml:"validation,omitempty" json:"validation,omitempty"`
	Step       string `yaml:"step,omitempty" json:"step,omitempty"`
	Rollup     string `yaml:"rollup,omitempty" json:"rollup,omitempty"`
	Teardown   string `yaml:"teardown,omitempty" json:"teardown,omitempty"`

	// Common properties
	Timeout   time.Duration  `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	DependsOn []string       `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Params    map[string]any `yaml:"params,omitempty" json:"params,omitempty"`
	Parallel  bool           `yaml:"parallel,omitempty" json:"parallel,omitempty"`
}

// ConditionConfig configures a skip condition.
type ConditionConfig struct {
	Expression string `yaml:"expression,omitempty" json:"expression,omitempty"`
	Env        string `yaml:"env,omitempty" json:"env,omitempty"`
	Flag       string `yaml:"flag,omitempty" json:"flag,omitempty"`
	Reason     string `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// ChaosProfile configures chaos engineering settings.
type ChaosProfile struct {
	Name        string              `yaml:"name,omitempty" json:"name,omitempty"`
	Description string              `yaml:"description,omitempty" json:"description,omitempty"`
	Network     NetworkChaosConfig  `yaml:"network,omitempty" json:"network,omitempty"`
	Resource    ResourceChaosConfig `yaml:"resource,omitempty" json:"resource,omitempty"`
	Custom      CustomChaosConfig   `yaml:"custom,omitempty" json:"custom,omitempty"`
}

// NetworkChaosConfig configures network chaos.
type NetworkChaosConfig struct {
	Latency    LatencyConfig    `yaml:"latency,omitempty" json:"latency,omitempty"`
	PacketLoss PacketLossConfig `yaml:"packet_loss,omitempty" json:"packet_loss,omitempty"`
	Partition  PartitionConfig  `yaml:"partition,omitempty" json:"partition,omitempty"`
}

// LatencyConfig configures latency injection.
type LatencyConfig struct {
	Enabled bool          `yaml:"enabled" json:"enabled"`
	Min     time.Duration `yaml:"min" json:"min"`
	Max     time.Duration `yaml:"max" json:"max"`
	Jitter  float64       `yaml:"jitter,omitempty" json:"jitter,omitempty"`
}

// PacketLossConfig configures packet loss.
type PacketLossConfig struct {
	Enabled    bool    `yaml:"enabled" json:"enabled"`
	Percentage float64 `yaml:"percentage" json:"percentage"`
}

// PartitionConfig configures network partitions.
type PartitionConfig struct {
	Enabled  bool          `yaml:"enabled" json:"enabled"`
	Duration time.Duration `yaml:"duration" json:"duration"`
	Targets  []string      `yaml:"targets,omitempty" json:"targets,omitempty"`
}

// ResourceChaosConfig configures resource chaos.
type ResourceChaosConfig struct {
	CPU    CPUChaosConfig    `yaml:"cpu,omitempty" json:"cpu,omitempty"`
	Memory MemoryChaosConfig `yaml:"memory,omitempty" json:"memory,omitempty"`
	IO     IOChaosConfig     `yaml:"io,omitempty" json:"io,omitempty"`
}

// CPUChaosConfig configures CPU chaos.
type CPUChaosConfig struct {
	Enabled    bool          `yaml:"enabled" json:"enabled"`
	Percentage int           `yaml:"percentage" json:"percentage"`
	Duration   time.Duration `yaml:"duration,omitempty" json:"duration,omitempty"`
}

// MemoryChaosConfig configures memory chaos.
type MemoryChaosConfig struct {
	Enabled    bool          `yaml:"enabled" json:"enabled"`
	Percentage int           `yaml:"percentage" json:"percentage"`
	Duration   time.Duration `yaml:"duration,omitempty" json:"duration,omitempty"`
}

// IOChaosConfig configures I/O chaos.
type IOChaosConfig struct {
	Enabled    bool          `yaml:"enabled" json:"enabled"`
	Percentage int           `yaml:"percentage" json:"percentage"`
	Duration   time.Duration `yaml:"duration,omitempty" json:"duration,omitempty"`
}

// CustomChaosConfig configures custom chaos operations.
type CustomChaosConfig struct {
	Before []CustomHookConfig `yaml:"before,omitempty" json:"before,omitempty"`
	During []CustomHookConfig `yaml:"during,omitempty" json:"during,omitempty"`
	After  []CustomHookConfig `yaml:"after,omitempty" json:"after,omitempty"`
}

// CustomHookConfig configures a custom chaos hook.
type CustomHookConfig struct {
	Name    string            `yaml:"name" json:"name"`
	Command string            `yaml:"command,omitempty" json:"command,omitempty"`
	Args    []string          `yaml:"args,omitempty" json:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
}

// MockProfile configures mocking behavior.
type MockProfile struct {
	Name        string       `yaml:"name,omitempty" json:"name,omitempty"`
	Description string       `yaml:"description,omitempty" json:"description,omitempty"`
	Services    []MockConfig `yaml:"services" json:"services"`
}

// MockConfig configures a mock service.
type MockConfig struct {
	Name        string             `yaml:"name" json:"name"`
	Type        string             `yaml:"type" json:"type"`
	Rules       []MockRuleConfig   `yaml:"rules,omitempty" json:"rules,omitempty"`
	Fallback    MockFallbackConfig `yaml:"fallback,omitempty" json:"fallback,omitempty"`
	Passthrough bool               `yaml:"passthrough,omitempty" json:"passthrough,omitempty"`
}

// MockRuleConfig configures a mock rule.
type MockRuleConfig struct {
	Match    MockMatchConfig    `yaml:"match" json:"match"`
	Response MockResponseConfig `yaml:"response" json:"response"`
	Delay    time.Duration      `yaml:"delay,omitempty" json:"delay,omitempty"`
}

// MockMatchConfig configures mock matching.
type MockMatchConfig struct {
	Method   string            `yaml:"method,omitempty" json:"method,omitempty"`
	Path     string            `yaml:"path,omitempty" json:"path,omitempty"`
	Headers  map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Body     string            `yaml:"body,omitempty" json:"body,omitempty"`
	BodyJSON map[string]any    `yaml:"body_json,omitempty" json:"body_json,omitempty"`
}

// MockResponseConfig configures mock response.
type MockResponseConfig struct {
	Status  int               `yaml:"status" json:"status"`
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Body    string            `yaml:"body,omitempty" json:"body,omitempty"`
	File    string            `yaml:"file,omitempty" json:"file,omitempty"`
}

// MockFallbackConfig configures mock fallback behavior.
type MockFallbackConfig struct {
	Action string `yaml:"action" json:"action"` // "error", "passthrough", "default"
	Status int    `yaml:"status,omitempty" json:"status,omitempty"`
	Body   string `yaml:"body,omitempty" json:"body,omitempty"`
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
	Definitions map[string]FlagDefinition `yaml:"definitions,omitempty" json:"definitions,omitempty"`
	Defaults    map[string]any            `yaml:"defaults,omitempty" json:"defaults,omitempty"`
}

// FlagDefinition defines a flag.
type FlagDefinition struct {
	Type        string   `yaml:"type" json:"type"` // bool, string, int, float, []string
	Default     any      `yaml:"default,omitempty" json:"default,omitempty"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool     `yaml:"required,omitempty" json:"required,omitempty"`
	Choices     []string `yaml:"choices,omitempty" json:"choices,omitempty"`
}

// OptionConfig configures a named option.
type OptionConfig struct {
	Description string         `yaml:"description,omitempty" json:"description,omitempty"`
	Flags       map[string]any `yaml:"flags,omitempty" json:"flags,omitempty"`
	Middleware  []string       `yaml:"middleware,omitempty" json:"middleware,omitempty"`
	Tags        []string       `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// BundlesConfig configures reusable bundles.
type BundlesConfig struct {
	Infrastructure map[string][]string `yaml:"infrastructure,omitempty" json:"infrastructure,omitempty"`
	Flags          map[string][]string `yaml:"flags,omitempty" json:"flags,omitempty"`
	Options        map[string][]string `yaml:"options,omitempty" json:"options,omitempty"`
	Middleware     map[string][]string `yaml:"middleware,omitempty" json:"middleware,omitempty"`
}

// SecretsConfig configures secret management.
type SecretsConfig struct {
	Provider      string            `yaml:"provider" json:"provider"`
	Path          string            `yaml:"path,omitempty" json:"path,omitempty"`
	Mapping       map[string]string `yaml:"mapping,omitempty" json:"mapping,omitempty"`
	FallbackToEnv bool              `yaml:"fallback_to_env,omitempty" json:"fallback_to_env,omitempty"`
	Vault         *VaultConfig      `yaml:"vault,omitempty" json:"vault,omitempty"`
}

// VaultConfig configures HashiCorp Vault.
type VaultConfig struct {
	Address string `yaml:"address" json:"address"`
	Token   string `yaml:"token,omitempty" json:"token,omitempty"`
	Path    string `yaml:"path,omitempty" json:"path,omitempty"`
}

// ExecutionConfig configures test execution.
type ExecutionConfig struct {
	Parallelism      int           `yaml:"parallelism,omitempty" json:"parallelism,omitempty"`
	DefaultTimeout   time.Duration `yaml:"default_timeout,omitempty" json:"default_timeout,omitempty"`
	RetryConfig      RetryConfig   `yaml:"retry,omitempty" json:"retry,omitempty"`
	TeardownMode     string        `yaml:"teardown_mode,omitempty" json:"teardown_mode,omitempty"` // "always", "on_failure", "never"
	FailFast         bool          `yaml:"fail_fast,omitempty" json:"fail_fast,omitempty"`
	ShuffleScenarios bool          `yaml:"shuffle_scenarios,omitempty" json:"shuffle_scenarios,omitempty"`
}

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxRetries int           `yaml:"max_retries,omitempty" json:"max_retries,omitempty"`
	Backoff    BackoffConfig `yaml:"backoff,omitempty" json:"backoff,omitempty"`
}

// BackoffConfig configures backoff behavior.
type BackoffConfig struct {
	Type       string        `yaml:"type" json:"type"` // "constant", "exponential", "linear"
	Initial    time.Duration `yaml:"initial,omitempty" json:"initial,omitempty"`
	Max        time.Duration `yaml:"max,omitempty" json:"max,omitempty"`
	Multiplier float64       `yaml:"multiplier,omitempty" json:"multiplier,omitempty"`
	Jitter     bool          `yaml:"jitter,omitempty" json:"jitter,omitempty"`
}

// ResultsConfig configures results storage.
type ResultsConfig struct {
	Storage   StorageConfig   `yaml:"storage,omitempty" json:"storage,omitempty"`
	Reports   []ReportConfig  `yaml:"reports,omitempty" json:"reports,omitempty"`
	Retention RetentionConfig `yaml:"retention,omitempty" json:"retention,omitempty"`
}

// StorageConfig configures storage backend.
type StorageConfig struct {
	Type   string            `yaml:"type" json:"type"` // "file", "s3", "gcs", "database"
	Path   string            `yaml:"path,omitempty" json:"path,omitempty"`
	Bucket string            `yaml:"bucket,omitempty" json:"bucket,omitempty"`
	Region string            `yaml:"region,omitempty" json:"region,omitempty"`
	Config map[string]string `yaml:"config,omitempty" json:"config,omitempty"`
}

// ReportConfig configures report generation.
type ReportConfig struct {
	Format string `yaml:"format" json:"format"` // "junit", "json", "html", "markdown"
	Path   string `yaml:"path,omitempty" json:"path,omitempty"`
}

// RetentionConfig configures result retention.
type RetentionConfig struct {
	Days    int  `yaml:"days,omitempty" json:"days,omitempty"`
	Count   int  `yaml:"count,omitempty" json:"count,omitempty"`
	Cleanup bool `yaml:"cleanup,omitempty" json:"cleanup,omitempty"`
}

// NotificationsConfig configures notifications.
type NotificationsConfig struct {
	Slack   SlackNotificationConfig   `yaml:"slack,omitempty" json:"slack,omitempty"`
	Email   EmailNotificationConfig   `yaml:"email,omitempty" json:"email,omitempty"`
	Webhook WebhookNotificationConfig `yaml:"webhook,omitempty" json:"webhook,omitempty"`
}

// SlackNotificationConfig configures Slack notifications.
type SlackNotificationConfig struct {
	Enabled    bool     `yaml:"enabled" json:"enabled"`
	WebhookURL string   `yaml:"webhook_url,omitempty" json:"webhook_url,omitempty"`
	Channel    string   `yaml:"channel,omitempty" json:"channel,omitempty"`
	OnEvents   []string `yaml:"on_events,omitempty" json:"on_events,omitempty"` // "start", "complete", "failure"
}

// EmailNotificationConfig configures email notifications.
type EmailNotificationConfig struct {
	Enabled    bool     `yaml:"enabled" json:"enabled"`
	SMTPServer string   `yaml:"smtp_server,omitempty" json:"smtp_server,omitempty"`
	From       string   `yaml:"from,omitempty" json:"from,omitempty"`
	To         []string `yaml:"to,omitempty" json:"to,omitempty"`
	OnEvents   []string `yaml:"on_events,omitempty" json:"on_events,omitempty"`
}

// WebhookNotificationConfig configures webhook notifications.
type WebhookNotificationConfig struct {
	Enabled  bool              `yaml:"enabled" json:"enabled"`
	URL      string            `yaml:"url,omitempty" json:"url,omitempty"`
	Headers  map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	OnEvents []string          `yaml:"on_events,omitempty" json:"on_events,omitempty"`
}
