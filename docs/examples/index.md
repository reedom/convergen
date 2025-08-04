# Examples

Comprehensive examples demonstrating Convergen's capabilities in real-world scenarios, from simple use cases to complex enterprise patterns.

## What You'll Find Here

### 🏭 **[Real-World Usage](real-world.md)**
Production-ready examples and patterns:

- Domain-to-storage layer conversions
- API request/response transformations
- Microservice boundary mappings
- Event sourcing and CQRS patterns
- Integration with popular frameworks

### 🔧 **[Generics Support](generics.md)**
Modern Go generics integration:

- Generic conversion functions
- Type parameter constraints
- Repository pattern implementations
- Collection transformations
- Type-safe wrapper conversions

### 🌐 **[Framework Integrations](integrations.md)**
Working with popular Go frameworks:

- Echo, Gin, Fiber web frameworks
- GORM, SQLBoiler, Ent ORM integrations
- gRPC and Protocol Buffer patterns
- GraphQL schema mappings
- Testing framework integrations

## Featured Examples

### Domain-Driven Design

Transform between domain models and persistence layers:

```go
type Convergen interface {
    // Domain to persistence
    // :typecast
    // :map ID.String() ID
    // :map CreatedAt.Unix() Created
    UserToPersistence(*domain.User) *persistence.User
    
    // Persistence to domain
    // :conv uuid.Parse ID ID
    // :conv time.Unix Created CreatedAt
    PersistenceToUser(*persistence.User) (*domain.User, error)
}
```

### API Response Patterns

Clean API transformations with security considerations:

```go
type APIConvergen interface {
    // Public API response
    // :skip Password
    // :skip InternalNotes
    // :map CreatedAt.Format("2006-01-02T15:04:05Z07:00") CreatedAt
    UserToPublicAPI(*internal.User) *api.PublicUser
    
    // Admin API with full details
    // :map CreatedAt.Format("2006-01-02T15:04:05Z07:00") CreatedAt
    // :map UpdatedAt.Format("2006-01-02T15:04:05Z07:00") UpdatedAt
    UserToAdminAPI(*internal.User) *api.AdminUser
}
```

### Microservice Communication

Type-safe service boundary crossing:

```go
type ServiceConvergen interface {
    // Internal to external service
    // :conv proto.Marshal Data
    // :map ServiceVersion Version
    InternalToExternal(*internal.Request) (*external.Request, error)
    
    // External to internal
    // :conv proto.Unmarshal Data
    // :literal ProcessedAt time.Now()
    ExternalToInternal(*external.Response) (*internal.Response, error)
}
```

## Usage Patterns by Scenario

### 1. **Web Application Backend**

```go
// Controller layer conversions
type WebConvergen interface {
    // Request binding
    // :typecast
    // :literal CreatedAt time.Now()
    RequestToModel(*api.CreateUserRequest) *model.User
    
    // Response formatting
    // :skip Password
    // :stringer Status
    ModelToResponse(*model.User) *api.UserResponse
}
```

### 2. **Data Pipeline Processing**

```go
// ETL pipeline transformations
type PipelineConvergen interface {
    // Raw data ingestion
    // :conv parseTimestamp Timestamp
    // :conv normalizeEmail Email
    RawToProcessed(*raw.Record) (*processed.Record, error)
    
    // Analytics aggregation
    // :map Sum Total
    // :literal ProcessedAt time.Now()
    ProcessedToAnalytics(*processed.Record) *analytics.Metric
}
```

### 3. **Event-Driven Architecture**

```go
// Event transformation
type EventConvergen interface {
    // Domain events to message bus
    // :conv json.Marshal Payload
    // :map EventType Type
    DomainToMessage(*domain.Event) (*message.Event, error)
    
    // Message bus to domain
    // :conv json.Unmarshal Payload
    MessageToDomain(*message.Event) (*domain.Event, error)
}
```

## Code Organization Strategies

### Single-Purpose Converters

Organize by business domain:

```
internal/
├── user/
│   ├── converter.go        // User-specific conversions
│   └── converter.gen.go    // Generated code
├── order/
│   ├── converter.go        // Order-specific conversions  
│   └── converter.gen.go    // Generated code
└── shared/
    ├── converter.go        // Cross-domain conversions
    └── converter.gen.go    // Generated code
```

### Layer-Based Converters

Organize by application layer:

```
internal/
├── api/
│   ├── converter.go        // API layer conversions
│   └── converter.gen.go
├── service/
│   ├── converter.go        // Service layer conversions
│   └── converter.gen.go
└── repository/
    ├── converter.go        // Data layer conversions
    └── converter.gen.go
```

### Multi-Interface Pattern

Multiple interfaces in one file:

```go
//go:build convergen

package converters

// :convergen
type UserConvergen interface {
    // :recv u
    ToAPI(*User) *api.User
    
    // :recv u
    ToStorage(*User) *storage.User
}

// :convergen
type OrderConvergen interface {
    // :typecast
    ToAPI(*Order) *api.Order
    
    // :conv encrypt.Data Data
    ToStorage(*Order) (*storage.Order, error)
}
```

## Testing Generated Code

### Unit Testing Patterns

```go
func TestUserConversions(t *testing.T) {
    user := &domain.User{
        ID:        uuid.New(),
        Email:     "test@example.com",
        CreatedAt: time.Now(),
    }
    
    // Test domain to API conversion
    apiUser := UserToAPI(user)
    assert.Equal(t, user.Email, apiUser.Email)
    assert.NotEmpty(t, apiUser.CreatedAt)
    
    // Test storage conversion
    storageUser := UserToStorage(user)
    assert.Equal(t, user.ID.String(), storageUser.ID)
    assert.Equal(t, user.CreatedAt.Unix(), storageUser.Created)
}
```

### Integration Testing

```go
func TestAPIIntegration(t *testing.T) {
    // Create test data
    user := createTestUser()
    
    // Test full pipeline
    apiReq := &api.CreateUserRequest{Email: user.Email}
    model := RequestToModel(apiReq)
    stored := ModelToStorage(model)
    retrieved := StorageToModel(stored)
    response := ModelToResponse(retrieved)
    
    // Verify round-trip integrity
    assert.Equal(t, user.Email, response.Email)
}
```

## Performance Considerations

### Allocation Optimization

```go
// Pre-allocate slices for better performance
type OptimizedConvergen interface {
    // :literal Items make([]*api.Item, 0, len(src.Items))
    // :conv convertItems Items
    OrderToAPI(*model.Order) *api.Order
}

func convertItems(items []*model.Item) []*api.Item {
    result := make([]*api.Item, len(items))
    for i, item := range items {
        result[i] = ItemToAPI(item)
    }
    return result
}
```

### Benchmarking

```go
func BenchmarkUserConversion(b *testing.B) {
    user := createTestUser()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = UserToAPI(user)
    }
}
```

## Next Steps

Ready to implement these patterns?

1. **Start with [Real-World Usage](real-world.md)** for production patterns
2. **Explore [Generics Support](generics.md)** for type-safe solutions
3. **Check [Framework Integrations](integrations.md)** for your tech stack
4. **Review [Best Practices](../guide/best-practices.md)** for team guidelines

## Additional Resources

- **Advanced patterns**: [Advanced Usage](../guide/advanced-usage.md)
- **Performance optimization**: [Performance Guide](../guide/performance.md)
- **Troubleshooting**: [Common Issues](../troubleshooting/common-issues.md)