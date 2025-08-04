# Troubleshooting

Having issues with Convergen? This section provides comprehensive troubleshooting guidance, from common issues to advanced debugging techniques.

## What You'll Find Here

### 🔧 **[Common Issues](common-issues.md)**
Frequently encountered problems and solutions:

- Installation and setup issues
- Generation failures and error messages
- Type compatibility problems
- Performance issues and optimization
- Integration problems with build systems

### 🐛 **[Debugging Guide](debugging.md)**
Advanced debugging techniques:

- Enabling debug logging and verbose output
- Understanding error messages and stack traces
- Debugging generated code issues
- Parser and compilation problems
- Runtime behavior analysis

### 📈 **[Migration Guide](migration.md)**
Upgrading between Convergen versions:

- Migration from v7 to v8
- Breaking changes and compatibility
- Performance improvements and new features
- Deprecation notices and replacements
- Upgrade strategies for large codebases

## Quick Diagnosis

### Is Your Issue Here?

=== "Generation Fails"

    **Symptoms**: Code generation produces errors or no output
    
    **Quick Checks**:
    - Verify Go version (1.21+ required for v8)
    - Check interface syntax and annotations
    - Ensure proper import statements
    - Validate file permissions and paths
    
    **Next Steps**: [Common Issues - Generation Problems](common-issues.md#generation-problems)

=== "Generated Code Won't Compile"

    **Symptoms**: Generated code has compilation errors
    
    **Quick Checks**:
    - Verify all imported types are accessible
    - Check for circular dependencies
    - Validate annotation syntax
    - Ensure converter functions exist
    
    **Next Steps**: [Debugging Guide - Compilation Issues](debugging.md#compilation-issues)

=== "Runtime Errors"

    **Symptoms**: Generated code compiles but fails at runtime
    
    **Quick Checks**:
    - Verify nil pointer handling
    - Check type conversions and casts
    - Validate converter function signatures
    - Review error handling patterns
    
    **Next Steps**: [Debugging Guide - Runtime Issues](debugging.md#runtime-issues)

=== "Performance Problems"

    **Symptoms**: Slow generation or runtime performance
    
    **Quick Checks**:
    - Update to v8 for parser improvements
    - Review complex annotation patterns
    - Check for inefficient converter functions
    - Analyze generated code patterns
    
    **Next Steps**: [Common Issues - Performance](common-issues.md#performance-issues)

## Error Message Reference

### Generation Errors

| Error Pattern | Likely Cause | Solution |
|---------------|--------------|----------|
| `interface not found` | Interface naming or visibility | Check interface name and exports |
| `cannot resolve type` | Import or type visibility issues | Verify imports and type accessibility |
| `annotation syntax error` | Malformed annotation | Review annotation syntax |
| `converter function not found` | Missing custom converter | Implement referenced converter function |

### Compilation Errors

| Error Pattern | Likely Cause | Solution |
|---------------|--------------|----------|
| `undefined: TypeName` | Missing import or type | Add required imports |
| `cannot convert` | Type incompatibility | Use `:typecast` or custom converter |
| `too many arguments` | Function signature mismatch | Review method signature and annotations |
| `not enough arguments` | Missing required parameters | Check converter function signatures |

## Self-Diagnosis Checklist

Before seeking help, run through this checklist:

### ✅ **Environment Check**

- [ ] Go version 1.21 or later installed
- [ ] Convergen properly installed or accessible
- [ ] Project uses Go modules (`go.mod` exists)
- [ ] Working directory is correct

### ✅ **Code Structure Check**

- [ ] Interface properly defined and exported
- [ ] All referenced types are imported and accessible
- [ ] Annotation syntax follows documented format
- [ ] Custom converter functions exist and are accessible

### ✅ **Generation Process Check**

- [ ] `go:generate` directive is correct
- [ ] File permissions allow reading/writing
- [ ] No conflicting generated files exist
- [ ] Build tags are properly configured

### ✅ **Output Validation Check**

- [ ] Generated code file exists and is readable
- [ ] Generated code compiles without errors
- [ ] Generated functions have expected signatures
- [ ] Runtime behavior matches expectations

## Getting Help

### Information to Provide

When reporting issues, include:

1. **Environment Details**
   ```bash
   go version
   # Output: go version go1.21.5 darwin/amd64
   ```

2. **Convergen Version**
   ```bash
   go run github.com/reedom/convergen@latest -version
   # Or check go.mod for version
   ```

3. **Minimal Reproduction Case**
   ```go
   // Complete minimal example that reproduces the issue
   //go:build convergen
   package example
   
   //go:generate go run github.com/reedom/convergen@v8.0.3
   type Convergen interface {
       // Your problematic interface definition
   }
   ```

4. **Error Messages**
   ```
   # Complete error output including stack traces
   ```

5. **Expected vs Actual Behavior**
   - What you expected to happen
   - What actually happened
   - Any workarounds you've tried

### Where to Get Help

- **GitHub Issues**: [Report bugs and get support](https://github.com/reedom/convergen/issues)
- **GitHub Discussions**: [Ask questions and discuss usage](https://github.com/reedom/convergen/discussions)
- **Documentation**: Search this documentation site
- **Go Package Docs**: [API reference on pkg.go.dev](https://pkg.go.dev/github.com/reedom/convergen)

## Prevention Tips

### Code Review Checklist

When adding or modifying Convergen interfaces:

- [ ] **Annotations are properly formatted** and follow documented syntax
- [ ] **All referenced types are accessible** and properly imported
- [ ] **Custom converter functions exist** and have correct signatures
- [ ] **Error handling is appropriate** for the use case
- [ ] **Generated code is tested** and behaves as expected

### Development Workflow

1. **Start Simple**: Begin with basic conversions and add complexity gradually
2. **Test Early**: Generate and test code frequently during development
3. **Version Pin**: Use specific versions in production (`@v8.0.3` not `@latest`)
4. **Document Patterns**: Document custom annotation patterns for team use
5. **Review Generated Code**: Periodically review generated code for optimization opportunities

## Common Debugging Commands

```bash
# Test generation with verbose output
go generate -x ./...

# Manual generation with logging
convergen -log -print your-file.go

# Check generated code syntax
go build -o /dev/null ./...

# Verify imports and dependencies
go mod tidy
go mod verify

# Clean and regenerate
find . -name "*.gen.go" -delete
go generate ./...
```

## Next Steps

Based on your issue type:

- **Generation problems**: [Common Issues](common-issues.md)
- **Complex debugging**: [Debugging Guide](debugging.md)
- **Version upgrades**: [Migration Guide](migration.md)
- **Performance concerns**: [Performance Guide](../guide/performance.md)

Still stuck? Don't hesitate to [open an issue](https://github.com/reedom/convergen/issues/new) with the information from our checklist!