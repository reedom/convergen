# Real-World Usage

Production-ready examples showcasing Convergen's new struct literal generation and enhanced generics support in real applications.

## E-Commerce Platform Example

### Project Structure
```
ecommerce/
├── internal/
│   ├── domain/          # Business logic
│   │   ├── user.go
│   │   ├── product.go
│   │   └── order.go
│   └── persistence/     # Database models
│       ├── user.go
│       ├── product.go
│       └── order.go
├── api/
│   └── dto/             # API responses
│       ├── user.go
│       ├── product.go
│       └── order.go
└── pkg/
    └── converters/      # Cross-package converters
        ├── user_converter.go
        ├── product_converter.go
        └── order_converter.go
```

### Domain-to-Persistence Conversions

**Domain models:**
```go
// internal/domain/user.go
package domain

import (
    "time"
    "github.com/google/uuid"
)

type User struct {
    ID          uuid.UUID
    Email       string
    FirstName   string
    LastName    string
    CreatedAt   time.Time
    LastLoginAt *time.Time
    IsActive    bool
    Preferences UserPreferences
}

type UserPreferences struct {
    Theme            string
    NotificationsEnabled bool
    Language         string
}
```

**Persistence models:**
```go
// internal/persistence/user.go
package persistence

import "time"

type User struct {
    ID          string    `db:"id"`
    Email       string    `db:"email"`
    FirstName   string    `db:"first_name"`
    LastName    string    `db:"last_name"`
    CreatedAt   time.Time `db:"created_at"`
    LastLoginAt *time.Time `db:"last_login_at"`
    IsActive    bool      `db:"is_active"`
    Theme       string    `db:"theme"`
    Notifications bool    `db:"notifications_enabled"`
    Language    string    `db:"language"`
}
```

**Cross-package generic converter:**
```go
// pkg/converters/user_converter.go
//go:build convergen

package converters

//go:generate convergen -type UserConverter[domain.User,persistence.User] -imports domain=../../internal/domain,persistence=../../internal/persistence -struct-literal $GOFILE

type UserConverter[S, D any] interface {
    // Domain to persistence with struct literals
    // :struct-literal
    // :map ID.String() ID
    // :map Preferences.Theme Theme
    // :map Preferences.NotificationsEnabled Notifications
    // :map Preferences.Language Language
    DomainToPersistence(S) D

    // Persistence to domain with error handling (auto-fallback to assignment)
    // :struct-literal
    // :conv parseUUID ID ID
    // :conv buildPreferences src Preferences
    PersistenceToDomain(D) (S, error)
}

func parseUUID(id string) (uuid.UUID, error) {
    return uuid.Parse(id)
}

func buildPreferences(user persistence.User) domain.UserPreferences {
    return domain.UserPreferences{
        Theme:                user.Theme,
        NotificationsEnabled: user.Notifications,
        Language:            user.Language,
    }
}
```

**Generated code (struct literal):**
```go
func DomainToPersistence(src domain.User) persistence.User {
    return persistence.User{
        ID:          src.ID.String(),
        Email:       src.Email,
        FirstName:   src.FirstName,
        LastName:    src.LastName,
        CreatedAt:   src.CreatedAt,
        LastLoginAt: src.LastLoginAt,
        IsActive:    src.IsActive,
        Theme:       src.Preferences.Theme,
        Notifications: src.Preferences.NotificationsEnabled,
        Language:    src.Preferences.Language,
    }
}

// Falls back to assignment block due to error handling
func PersistenceToDomain(src persistence.User) (dst domain.User, err error) {
    dst = domain.User{}

    dst.ID, err = parseUUID(src.ID)
    if err != nil {
        return
    }

    dst.Email = src.Email
    dst.FirstName = src.FirstName
    dst.LastName = src.LastName
    dst.CreatedAt = src.CreatedAt
    dst.LastLoginAt = src.LastLoginAt
    dst.IsActive = src.IsActive
    dst.Preferences = buildPreferences(src)
    return
}
```

### API Response Transformations

**API DTOs:**
```go
// api/dto/user.go
package dto

import "time"

type UserResponse struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    FullName  string    `json:"full_name"`
    CreatedAt string    `json:"created_at"`
    IsActive  bool      `json:"is_active"`
    Profile   UserProfile `json:"profile"`
}

type UserProfile struct {
    Theme     string `json:"theme"`
    Language  string `json:"language"`
}

type UserListResponse struct {
    Users []UserResponse `json:"users"`
    Total int           `json:"total"`
    Page  int           `json:"page"`
    Size  int           `json:"size"`
}
```

**API converter:**
```go
// pkg/converters/api_converter.go
//go:build convergen

package converters

//go:generate convergen -type APIConverter[domain.User,dto.UserResponse] -imports domain=../../internal/domain,dto=../../api/dto -struct-literal $GOFILE

type APIConverter[S, D any] interface {
    // Domain to API response with struct literals
    // :struct-literal
    // :map ID.String() ID
    // :conv formatFullName src FullName
    // :map CreatedAt.Format("2006-01-02T15:04:05Z") CreatedAt
    // :conv buildProfile src Profile
    // Note: Sensitive fields automatically excluded
    DomainToAPI(S) D

    // Batch conversion for list endpoints
    // :conv convertUserList src
    ConvertUserList([]S, int, int, int) dto.UserListResponse
}

func formatFullName(user domain.User) string {
    return user.FirstName + " " + user.LastName
}

func buildProfile(user domain.User) dto.UserProfile {
    return dto.UserProfile{
        Theme:    user.Preferences.Theme,
        Language: user.Preferences.Language,
    }
}

func convertUserList(users []domain.User, total, page, size int) dto.UserListResponse {
    userResponses := make([]dto.UserResponse, len(users))
    for i, user := range users {
        userResponses[i] = DomainToAPI(user)
    }

    return dto.UserListResponse{
        Users: userResponses,
        Total: total,
        Page:  page,
        Size:  size,
    }
}
```

## Microservice Communication

### Order Processing Service

**Service boundary models:**
```go
// internal/domain/order.go
package domain

type Order struct {
    ID       string
    UserID   string
    Items    []OrderItem
    Status   OrderStatus
    Total    Money
    ShippingAddress Address
}

type OrderItem struct {
    ProductID string
    Quantity  int
    UnitPrice Money
}

type Money struct {
    Amount   int64  // cents
    Currency string
}

type OrderStatus int

const (
    OrderPending OrderStatus = iota
    OrderConfirmed
    OrderShipped
    OrderDelivered
    OrderCancelled
)

func (s OrderStatus) String() string {
    switch s {
    case OrderPending:
        return "pending"
    case OrderConfirmed:
        return "confirmed"
    case OrderShipped:
        return "shipped"
    case OrderDelivered:
        return "delivered"
    case OrderCancelled:
        return "cancelled"
    default:
        return "unknown"
    }
}
```

**Event models for message passing:**
```go
// pkg/events/order.go
package events

type OrderCreatedEvent struct {
    OrderID         string     `json:"order_id"`
    CustomerID      string     `json:"customer_id"`
    Items          []LineItem `json:"items"`
    StatusText     string     `json:"status"`
    TotalAmount    int64      `json:"total_amount"`
    TotalCurrency  string     `json:"total_currency"`
    ShippingStreet string     `json:"shipping_street"`
    ShippingCity   string     `json:"shipping_city"`
    ShippingZip    string     `json:"shipping_zip"`
}

type LineItem struct {
    SKU      string `json:"sku"`
    Quantity int    `json:"quantity"`
    Price    int64  `json:"price_cents"`
}
```

**Event conversion:**
```go
// pkg/converters/event_converter.go
//go:build convergen

package converters

//go:generate convergen -type EventConverter[domain.Order,events.OrderCreatedEvent] -imports domain=../../internal/domain,events=../events -struct-literal $GOFILE

type EventConverter[S, D any] interface {
    // Domain to event with struct literals
    // :struct-literal
    // :map ID OrderID
    // :map UserID CustomerID
    // :conv convertItems Items Items
    // :map Status.String() StatusText
    // :map Total.Amount TotalAmount
    // :map Total.Currency TotalCurrency
    // :map ShippingAddress.Street ShippingStreet
    // :map ShippingAddress.City ShippingCity
    // :map ShippingAddress.PostalCode ShippingZip
    DomainToEvent(S) D
}

func convertItems(items []domain.OrderItem) []events.LineItem {
    result := make([]events.LineItem, len(items))
    for i, item := range items {
        result[i] = events.LineItem{
            SKU:      item.ProductID,
            Quantity: item.Quantity,
            Price:    item.UnitPrice.Amount,
        }
    }
    return result
}
```

## CQRS and Event Sourcing

### Command and Query Models

**Command model:**
```go
// internal/commands/user.go
package commands

type CreateUserCommand struct {
    Email     string
    FirstName string
    LastName  string
    Password  string // Will be excluded from events
}

type UpdateUserCommand struct {
    ID        string
    FirstName *string
    LastName  *string
    Email     *string
}
```

**Event store models:**
```go
// internal/events/user.go
package events

import "time"

type UserCreatedEvent struct {
    UserID    string    `json:"user_id"`
    Email     string    `json:"email"`
    FirstName string    `json:"first_name"`
    LastName  string    `json:"last_name"`
    CreatedAt time.Time `json:"created_at"`
    Version   int       `json:"version"`
}

type UserUpdatedEvent struct {
    UserID    string     `json:"user_id"`
    Changes   UserChanges `json:"changes"`
    UpdatedAt time.Time  `json:"updated_at"`
    Version   int        `json:"version"`
}

type UserChanges struct {
    FirstName *string `json:"first_name,omitempty"`
    LastName  *string `json:"last_name,omitempty"`
    Email     *string `json:"email,omitempty"`
}
```

**CQRS converters:**
```go
// pkg/converters/cqrs_converter.go
//go:build convergen

package converters

//go:generate convergen -type CQRSConverter[commands.CreateUserCommand,events.UserCreatedEvent] -imports commands=../../internal/commands,events=../../internal/events -struct-literal $GOFILE

type CQRSConverter[S, D any] interface {
    // Command to event with struct literals
    // :struct-literal
    // :conv generateUserID src UserID
    // :literal CreatedAt time.Now()
    // :literal Version 1
    // :skip Password  // Sensitive data excluded from events
    CommandToCreatedEvent(S) D

    // Update command to event
    // :struct-literal
    // :map ID UserID
    // :conv buildChanges src Changes
    // :literal UpdatedAt time.Now()
    // :conv getNextVersion ID Version
    UpdateCommandToEvent(commands.UpdateUserCommand) events.UserUpdatedEvent
}

func generateUserID(cmd commands.CreateUserCommand) string {
    return uuid.New().String()
}

func buildChanges(cmd commands.UpdateUserCommand) events.UserChanges {
    return events.UserChanges{
        FirstName: cmd.FirstName,
        LastName:  cmd.LastName,
        Email:     cmd.Email,
    }
}

func getNextVersion(userID string) int {
    // Implementation would fetch current version from event store
    return getCurrentVersion(userID) + 1
}
```

## Framework Integration

### Gin HTTP Framework

**HTTP handlers with automatic conversion:**
```go
// api/handlers/user.go
package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "myproject/pkg/converters"
)

type UserHandler struct {
    userService UserService
}

func (h *UserHandler) GetUser(c *gin.Context) {
    userID := c.Param("id")

    user, err := h.userService.GetByID(userID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
        return
    }

    // Convert domain model to API response using generated converter
    response := converters.DomainToAPI(user)
    c.JSON(http.StatusOK, response)
}

func (h *UserHandler) CreateUser(c *gin.Context) {
    var request dto.CreateUserRequest
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Convert API request to domain model
    user, err := converters.RequestToDomain(request)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    createdUser, err := h.userService.Create(user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
        return
    }

    response := converters.DomainToAPI(createdUser)
    c.JSON(http.StatusCreated, response)
}
```

### gRPC Integration

**Protocol buffer definitions:**
```protobuf
// proto/user.proto
syntax = "proto3";

package user.v1;

message User {
    string id = 1;
    string email = 2;
    string first_name = 3;
    string last_name = 4;
    int64 created_at = 5;
    bool is_active = 6;
}

message CreateUserRequest {
    string email = 1;
    string first_name = 2;
    string last_name = 3;
    string password = 4;
}
```

**gRPC converter:**
```go
// pkg/converters/grpc_converter.go
//go:build convergen

package converters

//go:generate convergen -type GRPCConverter[domain.User,pb.User] -imports domain=../../internal/domain,pb=../../proto/user/v1 -struct-literal $GOFILE

type GRPCConverter[S, D any] interface {
    // Domain to protobuf with struct literals
    // :struct-literal
    // :map CreatedAt.Unix() CreatedAt
    DomainToProto(S) *D

    // Protobuf to domain with validation
    // :struct-literal
    // :conv validateProtoUser src
    ProtoToDomain(*D) (S, error)
}

func validateProtoUser(user *pb.User) (domain.User, error) {
    if user.Email == "" {
        return domain.User{}, errors.New("email is required")
    }

    return domain.User{
        ID:        user.Id,
        Email:     user.Email,
        FirstName: user.FirstName,
        LastName:  user.LastName,
        CreatedAt: time.Unix(user.CreatedAt, 0),
        IsActive:  user.IsActive,
    }, nil
}
```

## Performance-Critical Scenarios

### High-Throughput Data Processing

**Batch processing with struct literals:**
```go
// pkg/converters/batch_converter.go
//go:build convergen

package converters

//go:generate convergen -type BatchConverter[domain.Product,dto.ProductSummary] -imports domain=../../internal/domain,dto=../../api/dto -struct-literal $GOFILE

type BatchConverter[S, D any] interface {
    // Single conversion with struct literal (fastest)
    // :struct-literal
    // :map Price.Amount PriceAmount
    // :map Price.Currency Currency
    // :map Categories[0] PrimaryCategory
    ConvertProduct(S) D

    // Batch conversion optimized
    // :conv convertProductBatch src
    ConvertProductBatch([]S) []D
}

func convertProductBatch(products []domain.Product) []dto.ProductSummary {
    // Pre-allocate with exact capacity for performance
    result := make([]dto.ProductSummary, len(products))

    // Use generated single converter for optimal performance
    for i, product := range products {
        result[i] = ConvertProduct(product)
    }

    return result
}
```

**Memory pool optimization:**
```go
var productSummaryPool = sync.Pool{
    New: func() interface{} {
        return &dto.ProductSummary{}
    },
}

type OptimizedConverter[S, D any] interface {
    // :conv convertWithPool src
    ConvertOptimized(S) *D
}

func convertWithPool(product domain.Product) *dto.ProductSummary {
    summary := productSummaryPool.Get().(*dto.ProductSummary)

    // Reset and populate using struct literal approach
    *summary = dto.ProductSummary{
        ID:              product.ID,
        Name:            product.Name,
        PriceAmount:     product.Price.Amount,
        Currency:        product.Price.Currency,
        PrimaryCategory: product.Categories[0],
    }

    return summary
}

// Don't forget to return to pool
func ReturnProductSummary(summary *dto.ProductSummary) {
    productSummaryPool.Put(summary)
}
```

## Testing Patterns

### Comprehensive Testing with Table-Driven Tests

```go
// pkg/converters/user_converter_test.go
package converters

import (
    "testing"
    "time"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "myproject/internal/domain"
    "myproject/internal/persistence"
)

func TestDomainToPersistence(t *testing.T) {
    tests := []struct {
        name     string
        input    domain.User
        expected persistence.User
    }{
        {
            name: "complete user conversion",
            input: domain.User{
                ID:          uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
                Email:       "john@example.com",
                FirstName:   "John",
                LastName:    "Doe",
                CreatedAt:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
                LastLoginAt: nil,
                IsActive:    true,
                Preferences: domain.UserPreferences{
                    Theme:                "dark",
                    NotificationsEnabled: true,
                    Language:            "en",
                },
            },
            expected: persistence.User{
                ID:            "550e8400-e29b-41d4-a716-446655440000",
                Email:         "john@example.com",
                FirstName:     "John",
                LastName:      "Doe",
                CreatedAt:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
                LastLoginAt:   nil,
                IsActive:      true,
                Theme:         "dark",
                Notifications: true,
                Language:      "en",
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := DomainToPersistence(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func BenchmarkDomainToPersistence(b *testing.B) {
    user := domain.User{
        ID:        uuid.New(),
        Email:     "test@example.com",
        FirstName: "Test",
        LastName:  "User",
        CreatedAt: time.Now(),
        IsActive:  true,
        Preferences: domain.UserPreferences{
            Theme:                "light",
            NotificationsEnabled: false,
            Language:            "en",
        },
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = DomainToPersistence(user)
    }
}
```

## Summary

These real-world examples demonstrate how Convergen's new features enable:

1. **Struct Literal Generation** - Cleaner, more performant code for simple conversions
2. **Cross-Package Generics** - Type-safe conversions across module boundaries
3. **Intelligent Fallback** - Automatic detection when struct literals aren't suitable
4. **Enhanced CLI Integration** - Powerful `-type` and `-imports` flags for complex scenarios
5. **Production-Ready Patterns** - Error handling, validation, and performance optimization

The combination of these features makes Convergen ideal for modern Go applications with complex architectures, microservices, and performance requirements.

## Next Steps

- **[Advanced Usage](../guide/advanced-usage.md)** - Deep dive into complex patterns
- **[Generics Examples](generics.md)** - Comprehensive generics documentation
- **[Best Practices](../guide/best-practices.md)** - Team development guidelines
- **[Migration Guide](../troubleshooting/migration.md)** - Upgrade to new features
    UserToPublicAPI(*internal.User) *api.PublicUser
}
```

For more examples, see the complete [Examples section](index.md).
