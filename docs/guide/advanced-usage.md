# Advanced Usage

Advanced patterns and techniques for using Convergen in complex scenarios, including the new struct literal generation and enhanced generics support.

## Struct Literal Generation

**NEW in v8.1** - Convergen can generate functions using struct literal syntax for cleaner, more performant code.

### Automatic Detection

Convergen automatically detects when struct literal syntax is appropriate:

```go
type User struct {
    ID    int
    Name  string
    Email string
}

type UserDTO struct {
    ID    int64
    Name  string
    Email string
}

type Convergen interface {
    // Simple conversion - automatically uses struct literal
    Convert(*User) *UserDTO
}
```

**Generated code:**

```go
func Convert(src *User) *UserDTO {
    return &UserDTO{
        ID:    int64(src.ID),
        Name:  src.Name,
        Email: src.Email,
    }
}
```

### Manual Control

Use annotations to override automatic detection:

```go
type Convergen interface {
    // Force struct literal even for complex cases
    // :struct-literal
    // :typecast
    ForceStructLiteral(*User) *UserDTO

    // Force assignment block for debugging
    // :no-struct-literal
    // :typecast
    ForceAssignment(*User) *UserDTO
}
```

### Fallback Scenarios

Convergen automatically falls back to assignment blocks when struct literals are incompatible:

```go
type Convergen interface {
    // Falls back due to :style arg
    // :struct-literal
    // :style arg
    ModifyInPlace(*UserDTO, *User)

    // Falls back due to error handling
    // :struct-literal
    // :conv validateEmail Email
    ConvertWithValidation(*User) (*UserDTO, error)

    // Falls back due to preprocessing
    // :struct-literal
    // :preprocess authorize
    SecureConvert(*User) (*UserDTO, error)
}
```

**Fallback reasons logged:**
- `:style arg` annotation modifies passed argument
- Error handling with multiple return paths
- Preprocessing hooks requiring imperative execution
- Complex custom converter functions

### Performance Benefits

Struct literals offer several advantages:

```go
// Struct literal (faster)
func Convert(src *User) *UserDTO {
    return &UserDTO{
        ID:   int64(src.ID),
        Name: src.Name,
    }
}

// Assignment block (more flexible)
func Convert(src *User) (dst *UserDTO) {
    dst = &UserDTO{}
    dst.ID = int64(src.ID)
    dst.Name = src.Name
    return
}
```

**Benefits:**
- Reduced memory allocations
- Better compiler optimization
- Cleaner generated code
- Improved readability

## Enhanced Generics Support

**MAJOR UPDATE in v8.1** - Production-ready generics with cross-package support.

### Cross-Package Generic Conversions

Convert between types in different packages using the enhanced CLI:

```bash
# Generate cross-package generic converter
convergen -type 'UserConverter[models.User,dto.UserDTO]' \
          -imports 'models=./internal/models,dto=./api/dto' \
          -struct-literal \
          user_converter.go
```

**Input:**

```go
//go:build convergen

package converters

type UserConverter[S, D any] interface {
    // :typecast
    // :struct-literal
    Convert(S) D
    ConvertSlice([]S) []D
}
```

**Generated:**

```go
package converters

import (
    "myproject/internal/models"
    "myproject/api/dto"
)

func Convert(src models.User) dto.UserDTO {
    return dto.UserDTO{
        ID:        int64(src.ID),
        Name:      src.Name,
        Email:     src.Email,
        CreatedAt: src.CreatedAt,
    }
}

func ConvertSlice(src []models.User) []dto.UserDTO {
    dst := make([]dto.UserDTO, len(src))
    for i, item := range src {
        dst[i] = Convert(item)
    }
    return dst
}
```

### Generic Constraint Support

**Any Constraint:**

```go
type Converter[T, U any] interface {
    Convert(T) U
    ConvertPtr(*T) *U
}
```

**Comparable Constraint:**

```go
type IDConverter[T comparable] interface {
    // :map String() Value
    ConvertID(ID[T]) StringID
}

type ID[T comparable] struct {
    Value T
}

func (id ID[T]) String() string {
    return fmt.Sprintf("%v", id.Value)
}

type StringID struct {
    Value string
}
```

**Union Constraints:**

```go
type NumericConverter[T ~int | ~int32 | ~int64, U ~float32 | ~float64] interface {
    // :typecast
    ConvertNumber(T) U
}
```

**Interface Constraints:**

```go
type StringerConverter[T fmt.Stringer] interface {
    // :map String() StringValue
    ConvertStringer(T) StringDTO
}

type StringDTO struct {
    StringValue string
}
```

### Type Compatibility Checking

Convergen validates generic type compatibility:

```bash
# Type mismatch detection
$ convergen -type 'Converter[models.User,dto.Product]' \
           -imports 'models=./models,dto=./dto' \
           converter.go

Warning: Field mismatch detected between models.User and dto.Product
  - models.User.Name (string) -> no matching field in dto.Product
  - dto.Product.Price (float64) -> no matching field in models.User
Suggestion: Add explicit field mappings with :map annotations
```

## Complex Error Handling Patterns

### Multi-Stage Error Handling

```go
type Convergen interface {
    // :preprocess validateInput
    // :conv hashPassword Password
    // :conv validateEmail Email
    // :postprocess auditConversion
    SecureConvert(*User) (*UserDTO, error)
}

func validateInput(dst *UserDTO, src *User) error {
    if src.Email == "" {
        return errors.New("email is required")
    }
    return nil
}

func hashPassword(password string) (string, error) {
    if len(password) < 8 {
        return "", errors.New("password too short")
    }
    return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func validateEmail(email string) (string, error) {
    if !strings.Contains(email, "@") {
        return "", errors.New("invalid email format")
    }
    return strings.ToLower(email), nil
}

func auditConversion(dst *UserDTO, src *User) error {
    log.Printf("Converted user %d", src.ID)
    return nil
}
```

**Generated code:**

```go
func SecureConvert(src *User) (dst *UserDTO, err error) {
    dst = &UserDTO{}

    // Preprocess validation
    err = validateInput(dst, src)
    if err != nil {
        return
    }

    // Field conversions with error handling
    dst.ID = src.ID
    dst.Name = src.Name

    dst.Password, err = hashPassword(src.Password)
    if err != nil {
        return
    }

    dst.Email, err = validateEmail(src.Email)
    if err != nil {
        return
    }

    // Postprocess audit
    err = auditConversion(dst, src)
    if err != nil {
        return
    }

    return
}
```

### Error Aggregation

```go
type ValidationError struct {
    Field   string
    Message string
}

type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
    var msgs []string
    for _, err := range ve {
        msgs = append(msgs, fmt.Sprintf("%s: %s", err.Field, err.Message))
    }
    return strings.Join(msgs, "; ")
}

type Convergen interface {
    // :conv validateAllFields src
    ConvertWithValidation(*User) (*UserDTO, error)
}

func validateAllFields(src *User) (*UserDTO, error) {
    var errors ValidationErrors

    if src.Email == "" {
        errors = append(errors, ValidationError{"Email", "required"})
    }

    if len(src.Password) < 8 {
        errors = append(errors, ValidationError{"Password", "too short"})
    }

    if len(errors) > 0 {
        return nil, errors
    }

    return &UserDTO{
        ID:    src.ID,
        Name:  src.Name,
        Email: src.Email,
    }, nil
}
```

## Complex Nested Struct Conversions

### Nested Structure Mapping

```go
type Address struct {
    Street  string
    City    string
    Country string
}

type User struct {
    ID      int
    Name    string
    Address Address
    Tags    []string
}

type AddressDTO struct {
    StreetAddress string `json:"street_address"`
    CityName      string `json:"city_name"`
    CountryCode   string `json:"country_code"`
}

type UserDTO struct {
    ID       int64       `json:"id"`
    FullName string      `json:"full_name"`
    Location AddressDTO  `json:"location"`
    Labels   []string    `json:"labels"`
}

type Convergen interface {
    // :map Address.Street Location.StreetAddress
    // :map Address.City Location.CityName
    // :conv convertCountry Address.Country Location.CountryCode
    // :map Name FullName
    // :map Tags Labels
    ConvertNested(*User) *UserDTO
}

func convertCountry(country string) string {
    countryMap := map[string]string{
        "United States": "US",
        "United Kingdom": "UK",
        "Canada": "CA",
    }
    if code, ok := countryMap[country]; ok {
        return code
    }
    return country
}
```

### Collection Transformations

```go
type Order struct {
    ID    int
    Items []OrderItem
    Total float64
}

type OrderItem struct {
    ProductID int
    Quantity  int
    Price     float64
}

type OrderDTO struct {
    ID         int64          `json:"id"`
    LineItems  []LineItemDTO  `json:"line_items"`
    TotalPrice string         `json:"total_price"`
}

type LineItemDTO struct {
    Product  int64  `json:"product_id"`
    Qty      int    `json:"quantity"`
    Amount   string `json:"amount"`
}

type Convergen interface {
    // :conv convertItems Items LineItems
    // :conv formatPrice Total TotalPrice
    ConvertOrder(*Order) *OrderDTO

    // Helper for individual items
    // :map ProductID Product
    // :map Quantity Qty
    // :conv formatPrice Price Amount
    ConvertOrderItem(*OrderItem) *LineItemDTO
}

func convertItems(items []OrderItem) []LineItemDTO {
    result := make([]LineItemDTO, len(items))
    for i, item := range items {
        result[i] = ConvertOrderItem(&item)
    }
    return result
}

func formatPrice(price float64) string {
    return fmt.Sprintf("$%.2f", price)
}
```

## Multiple Interface Definitions

### Domain-Specific Converters

```go
// User domain converters
// :convergen
type UserConverters interface {
    // :recv u
    ToDTO(*User) *UserDTO

    // :recv u
    ToEntity(*User) *UserEntity
}

// Product domain converters
// :convergen
type ProductConverters interface {
    // :recv p
    // :struct-literal
    ToView(*Product) *ProductView

    // :recv p
    // :conv calculateDiscount Price DiscountedPrice
    ToDiscountedView(*Product) *ProductView
}

// Order processing converters
// :convergen
type OrderProcessors interface {
    // :preprocess validateOrder
    // :postprocess notifyOrderCreated
    ProcessOrder(*OrderRequest) (*Order, error)

    // :conv aggregateItems Items Summary
    SummarizeOrder(*Order) *OrderSummary
}
```

### Shared Utilities

```go
// Common conversion utilities
// :convergen
type CommonConverters interface {
    // Generic timestamp conversion
    // :map Unix() Timestamp
    ConvertTime(time.Time) *TimestampDTO

    // Generic pagination
    // :literal Page src.Page
    // :literal Size src.Size
    // :literal Total len(src.Items)
    ConvertPagination[T, U any](*PaginatedList[T]) *PaginatedResponse[U]
}

type PaginatedList[T any] struct {
    Items []T
    Page  int
    Size  int
}

type PaginatedResponse[U any] struct {
    Data  []U `json:"data"`
    Page  int `json:"page"`
    Size  int `json:"size"`
    Total int `json:"total"`
}

type TimestampDTO struct {
    Timestamp int64 `json:"timestamp"`
}
```

## Integration with Existing Codebases

### Gradual Migration Strategy

**Phase 1: Identify Conversion Points**

```go
// Existing manual conversion
func UserToDTO(user *User) *UserDTO {
    return &UserDTO{
        ID:    int64(user.ID),
        Name:  user.Name,
        Email: user.Email,
    }
}

// Replace with Convergen interface
type Convergen interface {
    // :typecast
    // :struct-literal
    UserToDTO(*User) *UserDTO
}
```

**Phase 2: Add Type Safety**

```go
// Before: Runtime errors possible
func ConvertUsers(users []User) []UserDTO {
    result := make([]UserDTO, len(users))
    for i, user := range users {
        // Manual field copying - error prone
        result[i] = UserDTO{
            ID:   int64(user.ID),
            Name: user.Name,
            // Missing Email field - runtime bug!
        }
    }
    return result
}

// After: Compile-time validation
type Convergen interface {
    // :typecast
    ConvertUser(*User) *UserDTO

    // :conv convertUserSlice src
    ConvertUsers([]User) []UserDTO
}

func convertUserSlice(users []User) []UserDTO {
    result := make([]UserDTO, len(users))
    for i, user := range users {
        result[i] = ConvertUser(&user)
    }
    return result
}
```

**Phase 3: Add Error Handling**

```go
// Enhanced with validation
type Convergen interface {
    // Basic conversion
    // :typecast
    ConvertUser(*User) *UserDTO

    // With validation
    // :conv validateUser src
    ConvertUserSafe(*User) (*UserDTO, error)
}

func validateUser(user *User) (*UserDTO, error) {
    if user.Email == "" {
        return nil, errors.New("email required")
    }

    return &UserDTO{
        ID:    int64(user.ID),
        Name:  user.Name,
        Email: user.Email,
    }, nil
}
```

### Testing Generated Code

```go
func TestUserConversion(t *testing.T) {
    tests := []struct {
        name     string
        input    *User
        expected *UserDTO
        wantErr  bool
    }{
        {
            name: "valid user",
            input: &User{
                ID:    1,
                Name:  "John Doe",
                Email: "john@example.com",
            },
            expected: &UserDTO{
                ID:    1,
                Name:  "John Doe",
                Email: "john@example.com",
            },
            wantErr: false,
        },
        {
            name: "invalid email",
            input: &User{
                ID:    2,
                Name:  "Jane Doe",
                Email: "", // Invalid
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ConvertUserSafe(tt.input)

            if tt.wantErr {
                assert.Error(t, err)
                return
            }

            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

## Performance Optimization

### Memory Pool Usage

```go
var userDTOPool = sync.Pool{
    New: func() interface{} {
        return &UserDTO{}
    },
}

type Convergen interface {
    // :conv convertWithPool src
    ConvertUserOptimized(*User) *UserDTO
}

func convertWithPool(user *User) *UserDTO {
    dto := userDTOPool.Get().(*UserDTO)
    // Reset fields
    *dto = UserDTO{}

    // Populate
    dto.ID = int64(user.ID)
    dto.Name = user.Name
    dto.Email = user.Email

    return dto
}

// Remember to return to pool when done
func ReturnUserDTO(dto *UserDTO) {
    userDTOPool.Put(dto)
}
```

### Batch Processing

```go
type Convergen interface {
    // :conv convertBatch src
    ConvertUsersBatch([]User) []UserDTO
}

func convertBatch(users []User) []UserDTO {
    // Pre-allocate with exact capacity
    result := make([]UserDTO, 0, len(users))

    // Process in chunks for better cache locality
    const chunkSize = 100
    for i := 0; i < len(users); i += chunkSize {
        end := i + chunkSize
        if end > len(users) {
            end = len(users)
        }

        for j := i; j < end; j++ {
            result = append(result, UserDTO{
                ID:    int64(users[j].ID),
                Name:  users[j].Name,
                Email: users[j].Email,
            })
        }
    }

    return result
}
```

## Best Practices Summary

1. **Use struct literals** for simple conversions - better performance and readability
2. **Leverage cross-package generics** for type-safe conversions across module boundaries
3. **Implement proper error handling** for conversions that can fail
4. **Test generated code thoroughly** - use table-driven tests for comprehensive coverage
5. **Consider performance implications** - use memory pools and batch processing for high-throughput scenarios
6. **Organize by domain boundaries** - separate converter interfaces by business domain
7. **Use explicit type parameters** in generics for better documentation and maintainability

## Next Steps

- **[Annotations Reference](annotations.md)** - Master all available annotations
- **[Best Practices](best-practices.md)** - Learn team development patterns
- **[Examples](../examples/real-world.md)** - See production use cases
- **[CLI Reference](../api/cli.md)** - Command-line options and integration
