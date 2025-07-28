package parser

import "time"

// ConfigOption defines a functional option for parser configuration
type ConfigOption func(*ParserConfig)

// NewDefaultParserConfig creates a default parser configuration
func NewDefaultParserConfig() *ParserConfig {
	return &ParserConfig{
		BuildTag:                "convergen",
		MaxConcurrentWorkers:    4,
		TypeResolutionTimeout:   30 * time.Second,
		CacheSize:               1000,
		EnableProgress:          true,
		EnableConcurrentLoading: false, // Disabled by default for compatibility
		EnableMethodConcurrency: false, // Disabled by default for compatibility
	}
}

// NewTestParserConfig creates a configuration optimized for testing
func NewTestParserConfig() *ParserConfig {
	return &ParserConfig{
		BuildTag:                "convergen",
		MaxConcurrentWorkers:    2,
		TypeResolutionTimeout:   5 * time.Second,
		CacheSize:               100,
		EnableProgress:          false, // Disable for testing
		EnableConcurrentLoading: false,
		EnableMethodConcurrency: false,
	}
}

// NewConcurrentParserConfig creates a configuration with concurrency enabled
func NewConcurrentParserConfig() *ParserConfig {
	config := NewDefaultParserConfig()
	config.EnableConcurrentLoading = true
	config.EnableMethodConcurrency = true
	return config
}

// NewParserConfigWithOptions creates a parser configuration with functional options
func NewParserConfigWithOptions(options ...ConfigOption) *ParserConfig {
	config := NewDefaultParserConfig()
	for _, option := range options {
		option(config)
	}
	return config
}

// Functional options for parser configuration

// WithBuildTag sets the build tag
func WithBuildTag(tag string) ConfigOption {
	return func(config *ParserConfig) {
		config.BuildTag = tag
	}
}

// WithMaxWorkers sets the maximum concurrent workers
func WithMaxWorkers(workers int) ConfigOption {
	return func(config *ParserConfig) {
		config.MaxConcurrentWorkers = workers
	}
}

// WithTimeout sets the type resolution timeout
func WithTimeout(timeout time.Duration) ConfigOption {
	return func(config *ParserConfig) {
		config.TypeResolutionTimeout = timeout
	}
}

// WithCacheSize sets the cache size
func WithCacheSize(size int) ConfigOption {
	return func(config *ParserConfig) {
		config.CacheSize = size
	}
}

// WithProgress enables or disables progress reporting
func WithProgress(enabled bool) ConfigOption {
	return func(config *ParserConfig) {
		config.EnableProgress = enabled
	}
}

// WithConcurrentLoading enables or disables concurrent loading
func WithConcurrentLoading(enabled bool) ConfigOption {
	return func(config *ParserConfig) {
		config.EnableConcurrentLoading = enabled
	}
}

// WithMethodConcurrency enables or disables method concurrency
func WithMethodConcurrency(enabled bool) ConfigOption {
	return func(config *ParserConfig) {
		config.EnableMethodConcurrency = enabled
	}
}

// WithConcurrency enables both concurrent loading and method concurrency
func WithConcurrency(enabled bool) ConfigOption {
	return func(config *ParserConfig) {
		config.EnableConcurrentLoading = enabled
		config.EnableMethodConcurrency = enabled
	}
}

// EnsureValidConfig ensures the configuration has valid values
func EnsureValidConfig(config *ParserConfig) *ParserConfig {
	if config == nil {
		return NewDefaultParserConfig()
	}

	// Validate and fix invalid values
	if config.MaxConcurrentWorkers <= 0 {
		config.MaxConcurrentWorkers = 4
	}
	if config.TypeResolutionTimeout <= 0 {
		config.TypeResolutionTimeout = 30 * time.Second
	}
	if config.CacheSize <= 0 {
		config.CacheSize = 1000
	}
	if config.BuildTag == "" {
		config.BuildTag = "convergen"
	}

	return config
}

// CloneConfig creates a deep copy of the parser configuration
func CloneConfig(config *ParserConfig) *ParserConfig {
	if config == nil {
		return NewDefaultParserConfig()
	}

	return &ParserConfig{
		BuildTag:                config.BuildTag,
		MaxConcurrentWorkers:    config.MaxConcurrentWorkers,
		TypeResolutionTimeout:   config.TypeResolutionTimeout,
		CacheSize:               config.CacheSize,
		EnableProgress:          config.EnableProgress,
		EnableConcurrentLoading: config.EnableConcurrentLoading,
		EnableMethodConcurrency: config.EnableMethodConcurrency,
	}
}
