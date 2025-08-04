# Configuration

Advanced configuration options for Convergen, including global settings, environment variables, and performance tuning.

## Coming Soon

This section will cover:

- Global configuration files
- Per-project settings
- Environment variable support
- Parser configuration options
- Performance tuning parameters

*This page is currently being developed with comprehensive configuration options.*

## Quick Configuration

### Parser Configuration (v8)

The v8 parser engine supports multiple strategies:

```go
// For programmatic usage
config := parser.NewConcurrentParserConfig()
config.WithTimeout(30 * time.Second)
config.WithConcurrency(4)

modernParser := parser.NewModernParser(config)
```

### Environment Variables

```bash
# Set log level
export CONVERGEN_LOG_LEVEL=debug

# Set parser strategy
export CONVERGEN_PARSER_STRATEGY=modern
```

### CLI Configuration

```bash
# Enable logging
convergen -log converter.go

# Set output location
convergen -out custom_output.go converter.go
```

For complete CLI options, see [CLI Commands](cli.md).

For integration patterns, see [go:generate Integration](go-generate.md).