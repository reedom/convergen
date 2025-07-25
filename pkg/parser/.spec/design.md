# Parser Package Design

This document outlines the design of the `pkg/parser` package, which transforms Go source code into domain models for the generation pipeline.

## Architecture Overview

The parser follows a multi-stage pipeline architecture with clear separation of concerns:

1. **Source Analysis**: AST parsing and basic structure extraction
2. **Annotation Processing**: Comment-based configuration parsing
3. **Type Resolution**: Comprehensive type analysis with generic support
4. **Domain Model Construction**: Transformation to domain entities
5. **Validation**: Model consistency and correctness verification
6. **Event Emission**: Publishing results to the generation pipeline

## Core Components

### Parser Coordinator

```go
// Parser orchestrates the parsing pipeline
type Parser struct {
    config        *Config
    eventBus      events.EventBus
    typeResolver  *TypeResolver
    annotationReg *AnnotationRegistry
    validator     *DomainValidator
    cache         *ParseCache
    logger        *zap.Logger
}

// Main parsing entry point with context support
func (p *Parser) Parse(ctx context.Context, sourcePath string) (*domain.GenerationResult, error) {
    // 1. Load and analyze source file
    pkg, err := p.loadPackage(ctx, sourcePath)
    if err != nil {
        return nil, fmt.Errorf("failed to load package: %w", err)
    }
    
    // 2. Discover converter interfaces
    interfaces, err := p.discoverInterfaces(ctx, pkg)
    if err != nil {
        return nil, fmt.Errorf("interface discovery failed: %w", err)
    }
    
    // 3. Process each interface
    methods := make([]*domain.Method, 0)
    for _, intf := range interfaces {
        interfaceMethods, err := p.processInterface(ctx, intf)
        if err != nil {
            return nil, fmt.Errorf("interface processing failed: %w", err)
        }
        methods = append(methods, interfaceMethods...)
    }
    
    // 4. Validate domain models
    if err := p.validator.ValidateMethods(ctx, methods); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    
    // 5. Generate base code
    baseCode, err := p.generateBaseCode(ctx, pkg, interfaces)
    if err != nil {
        return nil, fmt.Errorf("base code generation failed: %w", err)
    }
    
    // 6. Emit parse event
    event := &ParseEvent{
        Methods:  methods,
        BaseCode: baseCode,
        Context:  ctx,
    }
    
    if err := p.eventBus.Publish(ctx, event); err != nil {
        return nil, fmt.Errorf("failed to emit parse event: %w", err)
    }
    
    return &domain.GenerationResult{
        Methods:  methods,
        BaseCode: baseCode,
    }, nil
}
```

### Interface Discovery

```go
// InterfaceDiscoverer finds converter interfaces in source code
type InterfaceDiscoverer struct {
    typeInfo *types.Info
    fset     *token.FileSet
    logger   *zap.Logger
}

// DiscoverInterfaces finds all converter interfaces
func (d *InterfaceDiscoverer) DiscoverInterfaces(ctx context.Context, pkg *packages.Package) ([]*ConvergenInterface, error) {
    interfaces := make([]*ConvergenInterface, 0)
    
    // Walk through all files in package
    for _, file := range pkg.Syntax {
        for _, decl := range file.Decls {
            if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
                for _, spec := range genDecl.Specs {
                    if typeSpec, ok := spec.(*ast.TypeSpec); ok {
                        if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
                            if intf := d.checkConvergenInterface(ctx, typeSpec, interfaceType, genDecl.Doc); intf != nil {
                                interfaces = append(interfaces, intf)
                            }
                        }
                    }
                }
            }
        }
    }
    
    return interfaces, nil
}

// checkConvergenInterface determines if interface is a converter
func (d *InterfaceDiscoverer) checkConvergenInterface(ctx context.Context, spec *ast.TypeSpec, iface *ast.InterfaceType, doc *ast.CommentGroup) *ConvergenInterface {
    // Check if named "Convergen"
    if spec.Name.Name == "Convergen" {
        return d.createConvergenInterface(ctx, spec, iface, doc)
    }
    
    // Check for :convergen annotation
    if doc != nil {
        for _, comment := range doc.List {
            if strings.Contains(comment.Text, ":convergen") {
                return d.createConvergenInterface(ctx, spec, iface, doc)
            }
        }
    }
    
    return nil
}
```

### Annotation Processing System

```go
// AnnotationRegistry manages extensible annotation processing
type AnnotationRegistry struct {
    processors map[string]AnnotationProcessor
    validators []AnnotationValidator
    logger     *zap.Logger
}

// AnnotationProcessor handles specific annotation types
type AnnotationProcessor interface {
    Name() string
    Pattern() *regexp.Regexp
    Parse(ctx context.Context, comment string, context ProcessingContext) (AnnotationConfig, error)
    Validate(ctx context.Context, config AnnotationConfig, context ValidationContext) error
}

// Built-in annotation processors
type MatchAnnotationProcessor struct{}

func (p *MatchAnnotationProcessor) Parse(ctx context.Context, comment string, context ProcessingContext) (AnnotationConfig, error) {
    // Parse :match <algorithm> annotation
    matches := p.Pattern().FindStringSubmatch(comment)
    if len(matches) < 2 {
        return nil, fmt.Errorf("invalid :match annotation format")
    }
    
    algorithm := strings.TrimSpace(matches[1])
    if algorithm != "name" && algorithm != "none" {
        return nil, fmt.Errorf("invalid match algorithm: %s", algorithm)
    }
    
    return &MatchConfig{
        Algorithm: algorithm,
    }, nil
}

// Registry-based processing
func (r *AnnotationRegistry) ProcessComments(ctx context.Context, comments []*ast.Comment, context ProcessingContext) ([]AnnotationConfig, error) {
    configs := make([]AnnotationConfig, 0)
    
    for _, comment := range comments {
        text := strings.TrimPrefix(comment.Text, "//")
        text = strings.TrimSpace(text)
        
        if !strings.HasPrefix(text, ":") {
            continue
        }
        
        // Find matching processor
        for _, processor := range r.processors {
            if processor.Pattern().MatchString(text) {
                config, err := processor.Parse(ctx, text, context)
                if err != nil {
                    return nil, fmt.Errorf("failed to parse %s annotation: %w", processor.Name(), err)
                }
                configs = append(configs, config)
                break
            }
        }
    }
    
    // Validate annotation combinations
    if err := r.validateConfigs(ctx, configs, context); err != nil {
        return nil, fmt.Errorf("annotation validation failed: %w", err)
    }
    
    return configs, nil
}
```

### Type Resolution Engine

```go
// TypeResolver handles comprehensive type analysis
type TypeResolver struct {
    typeInfo  *types.Info
    cache     *TypeCache
    logger    *zap.Logger
}

// ResolveType converts Go types to domain types
func (r *TypeResolver) ResolveType(ctx context.Context, goType types.Type) (domain.Type, error) {
    // Check cache first
    if cached, ok := r.cache.Get(goType.String()); ok {
        return cached, nil
    }
    
    var domainType domain.Type
    var err error
    
    switch t := goType.(type) {
    case *types.Basic:
        domainType, err = r.resolveBasicType(ctx, t)
    case *types.Named:
        domainType, err = r.resolveNamedType(ctx, t)
    case *types.Struct:
        domainType, err = r.resolveStructType(ctx, t)
    case *types.Slice:
        domainType, err = r.resolveSliceType(ctx, t)
    case *types.Map:
        domainType, err = r.resolveMapType(ctx, t)
    case *types.Interface:
        domainType, err = r.resolveInterfaceType(ctx, t)
    case *types.Pointer:
        domainType, err = r.resolvePointerType(ctx, t)
    case *types.TypeParam:
        domainType, err = r.resolveTypeParam(ctx, t)
    default:
        err = fmt.Errorf("unsupported type: %T", t)
    }
    
    if err != nil {
        return nil, err
    }
    
    // Cache the result
    r.cache.Put(goType.String(), domainType)
    return domainType, nil
}

// resolveStructType handles struct types with field ordering
func (r *TypeResolver) resolveStructType(ctx context.Context, structType *types.Struct) (domain.Type, error) {
    fields := make([]domain.Field, structType.NumFields())
    
    for i := 0; i < structType.NumFields(); i++ {
        field := structType.Field(i)
        tag := structType.Tag(i)
        
        fieldType, err := r.ResolveType(ctx, field.Type())
        if err != nil {
            return nil, fmt.Errorf("failed to resolve field %s type: %w", field.Name(), err)
        }
        
        fields[i] = domain.Field{
            Name:     field.Name(),
            Type:     fieldType,
            Tags:     reflect.StructTag(tag),
            Position: i,  // Preserve field order
            Exported: field.Exported(),
        }
    }
    
    return &domain.StructType{
        Fields:  fields,
        Package: structType.String(), // Package info for cross-package support
    }, nil
}
```

### Method Processing

```go
// MethodProcessor converts interface methods to domain models
type MethodProcessor struct {
    typeResolver  *TypeResolver
    annotationReg *AnnotationRegistry
    validator     *MethodValidator
    logger        *zap.Logger
}

// ProcessMethod converts an interface method to domain model
func (p *MethodProcessor) ProcessMethod(ctx context.Context, method *types.Func, comments []*ast.Comment, intf *ConvergenInterface) (*domain.Method, error) {
    signature := method.Type().(*types.Signature)
    
    // Extract method signature info
    methodSig, err := p.extractSignature(ctx, method, signature)
    if err != nil {
        return nil, fmt.Errorf("failed to extract method signature: %w", err)
    }
    
    // Process annotations
    annotations, err := p.annotationReg.ProcessComments(ctx, comments, ProcessingContext{
        Method:    method,
        Interface: intf,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to process annotations: %w", err)
    }
    
    // Build method configuration
    config, err := p.buildMethodConfig(ctx, annotations, intf.Config)
    if err != nil {
        return nil, fmt.Errorf("failed to build method config: %w", err)
    }
    
    // Create domain method
    domainMethod := &domain.Method{
        Name:       method.Name(),
        SourceType: methodSig.SourceType,
        DestType:   methodSig.DestType,
        Config:     config,
        Signature:  methodSig,
    }
    
    // Generate field mappings (will be done in planner)
    // This is kept minimal in parser - detailed mapping is planner's job
    
    return domainMethod, nil
}
```

### Base Code Generation

```go
// BaseCodeGenerator creates clean source without converter interfaces
type BaseCodeGenerator struct {
    fset   *token.FileSet
    logger *zap.Logger
}

// GenerateBaseCode removes converter interfaces and replaces with markers
func (g *BaseCodeGenerator) GenerateBaseCode(ctx context.Context, pkg *packages.Package, interfaces []*ConvergenInterface) (string, error) {
    var buf bytes.Buffer
    
    for _, file := range pkg.Syntax {
        // Clone AST for modification
        fileCopy := g.cloneFile(file)
        
        // Remove converter interfaces and add markers
        g.replaceInterfaces(fileCopy, interfaces)
        
        // Format and write to buffer
        if err := format.Node(&buf, g.fset, fileCopy); err != nil {
            return "", fmt.Errorf("failed to format file: %w", err)
        }
    }
    
    return buf.String(), nil
}
```

## Event Integration

### Parse Event Definition

```go
// ParseEvent represents successful parsing completion
type ParseEvent struct {
    BaseEvent
    Methods  []*domain.Method
    BaseCode string
    Metrics  ParseMetrics
}

// ParseMetrics track parsing performance
type ParseMetrics struct {
    ParseDurationMS     int64
    InterfacesFound     int
    MethodsProcessed    int
    AnnotationsProcessed int
    TypesResolved       int
    CacheHitRate        float64
}
```

## Caching Strategy

### Multi-Level Caching

```go
// ParseCache provides multi-level caching
type ParseCache struct {
    typeCache       *TypeCache
    methodCache     *MethodCache
    annotationCache *AnnotationCache
    packageCache    *PackageCache
}

// TypeCache caches resolved type information
type TypeCache struct {
    cache map[string]domain.Type
    mutex sync.RWMutex
    stats CacheStats
}

// Concurrent-safe caching with LRU eviction
func (c *TypeCache) Get(key string) (domain.Type, bool) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    
    if value, ok := c.cache[key]; ok {
        c.stats.Hits++
        return value, true
    }
    
    c.stats.Misses++
    return nil, false
}
```

## Error Handling Strategy

### Rich Error Context

```go
// ParseError provides detailed parsing error information
type ParseError struct {
    Code     ErrorCode
    Message  string
    File     string
    Line     int
    Column   int
    Context  string
    Cause    error
    Suggestions []string
}

// Error aggregation for batch reporting
type ErrorCollector struct {
    errors []ParseError
    mutex  sync.Mutex
}

func (c *ErrorCollector) Collect(err ParseError) {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    c.errors = append(c.errors, err)
}
```

This design provides a robust, extensible parser that efficiently transforms Go source code into domain models while supporting concurrent processing, comprehensive error handling, and integration with the event-driven pipeline architecture.
