# Framework Integrations

Examples of using Convergen with popular Go frameworks and libraries.

## Coming Soon

This section will cover:

- Echo, Gin, Fiber web frameworks
- GORM, SQLBoiler, Ent ORM integrations
- gRPC and Protocol Buffer patterns
- GraphQL schema mappings
- Testing framework integrations

*Framework integration examples are currently being developed.*

## Quick Examples

### Web Framework Integration

```go
// Echo framework
func (h *Handler) CreateUser(c echo.Context) error {
    var req api.CreateUserRequest
    if err := c.Bind(&req); err != nil {
        return err
    }
    
    user := RequestToUser(&req)  // Generated converter
    // ... business logic
    
    response := UserToResponse(user)  // Generated converter
    return c.JSON(200, response)
}
```

### ORM Integration

```go
// GORM integration
type Convergen interface {
    // :typecast
    // :map ID.String() ID
    DomainToGORM(*domain.User) *gorm.User
    
    // :conv uuid.Parse ID ID
    GORMToDomain(*gorm.User) (*domain.User, error)
}
```

### gRPC Integration

```go
// Protocol Buffers
type Convergen interface {
    // :conv proto.Marshal Data
    ModelToProto(*internal.Model) (*pb.Model, error)
    
    // :conv proto.Unmarshal Data
    ProtoToModel(*pb.Model) (*internal.Model, error)
}
```

For more examples, see the complete [Examples section](index.md).