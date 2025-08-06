# Basic Examples

This guide demonstrates common Convergen usage patterns through practical examples. Each example builds on the previous ones, introducing new concepts progressively.

## Example 1: Simple Field Matching

**Scenario:** Convert between identical structures

```go
// Source type
type User struct {
    ID   int
    Name string
    Email string
}

// Destination type  
type UserDTO struct {
    ID   int
    Name string
    Email string
}

// Converter interface
//go:generate go run github.com/reedom/convergen@v8.0.3
type Convergen interface {
    UserToDTO(*User) *UserDTO
}
```

**Generated code:**

```go
func UserToDTO(src *User) (dst *UserDTO) {
    dst = &UserDTO{}
    dst.ID = src.ID
    dst.Name = src.Name
    dst.Email = src.Email
    return
}
```

**Key Points:**
- ✅ **Automatic matching** by field name and type
- ✅ **Zero configuration** needed for identical fields
- ✅ **Type-safe** compilation validation

## Example 2: Skipping Sensitive Fields

**Scenario:** Exclude sensitive data from API responses

```go
type User struct {
    ID       int
    Name     string
    Email    string
    Password string    // Sensitive - exclude from API
    APIKey   string    // Sensitive - exclude from API
}

type PublicUserDTO struct {
    ID    int
    Name  string
    Email string
}

type Convergen interface {
    // Skip sensitive fields
    // :skip Password
    // :skip APIKey
    UserToPublic(*User) *PublicUserDTO
}
```

**Generated code:**

```go
func UserToPublic(src *User) (dst *PublicUserDTO) {
    dst = &PublicUserDTO{}
    dst.ID = src.ID
    dst.Name = src.Name
    dst.Email = src.Email
    // Password and APIKey are skipped
    return
}
```

**Key Points:**
- ✅ **Security by design** - sensitive fields explicitly excluded
- ✅ **Multiple skip rules** supported
- ✅ **Compile-time validation** prevents accidental exposure

## Example 3: Field Name Mapping

**Scenario:** Convert between fields with different names

```go
type User struct {
    ID        int
    FirstName string
    LastName  string
    Email     string
}

type UserResponse struct {
    UserID    int    // Different name
    FirstName string // Direct mapping
    LastName  string // Direct mapping
    Email     string
}

type Convergen interface {
    // Map different field names
    // :map ID UserID
    UserToResponse(*User) *UserResponse
}
```

**Generated code:**

```go
func UserToResponse(src *User) (dst *UserResponse) {
    dst = &UserResponse{}
    dst.UserID = src.ID
    dst.FirstName = src.FirstName
    dst.LastName = src.LastName
    dst.Email = src.Email
    return
}
```

**Key Points:**
- ✅ **Flexible field mapping** with `:map` annotation
- ✅ **Direct field name mapping** for different field names
- ✅ **Explicit control** over field assignments

## Example 4: Type Conversion

**Scenario:** Handle type differences between source and destination

```go
type User struct {
    ID     int        // int type
    Age    int        // int type
    Status UserStatus // Custom enum type
}

type UserStatus int
const (
    StatusInactive UserStatus = iota
    StatusActive
    StatusSuspended
)

func (s UserStatus) String() string {
    switch s {
    case StatusActive: return "active"
    case StatusSuspended: return "suspended"
    default: return "inactive"
    }
}

type UserDTO struct {
    ID     int64  // Different numeric type
    Age    string // String representation
    Status string // String representation
}

type Convergen interface {
    // Enable type casting and string conversion
    // :typecast
    // :stringer
    // :conv strconv.Itoa Age Age
    UserToDTO(*User) *UserDTO
}
```

**Generated code:**

```go
func UserToDTO(src *User) (dst *UserDTO) {
    dst = &UserDTO{}
    dst.ID = int64(src.ID)              // Type cast applied
    dst.Age = strconv.Itoa(src.Age)     // Custom converter
    dst.Status = src.Status.String()    // String() method used
    return
}
```

**Key Points:**
- ✅ **Automatic type casting** with `:typecast`
- ✅ **String conversion** with `:stringer`
- ✅ **Custom converters** for complex transformations

## Example 5: Error Handling

**Scenario:** Handle conversions that can fail

```go
import (
    "errors"
    "strconv"
    "strings"
)

type User struct {
    ID    int
    Email string
    Age   string // String that needs parsing
}

type UserEntity struct {
    ID    int
    Email string
    Age   int    // Parsed integer
}

type Convergen interface {
    // Use converters that can return errors
    // :conv validateEmail Email
    // :conv parseAge Age
    UserToEntity(*User) (*UserEntity, error)
}

func validateEmail(email string) (string, error) {
    if !strings.Contains(email, "@") {
        return "", errors.New("invalid email format")
    }
    return email, nil
}

func parseAge(ageStr string) (int, error) {
    age, err := strconv.Atoi(ageStr)
    if err != nil {
        return 0, errors.New("invalid age format")
    }
    if age < 0 || age > 150 {
        return 0, errors.New("age out of valid range")
    }
    return age, nil
}
```

**Generated code:**

```go
func UserToEntity(src *User) (dst *UserEntity, err error) {
    dst = &UserEntity{}
    dst.ID = src.ID
    dst.Email, err = validateEmail(src.Email)
    if err != nil {
        return
    }
    dst.Age, err = parseAge(src.Age)
    if err != nil {
        return
    }
    return
}
```

**Key Points:**
- ✅ **Error propagation** handled automatically
- ✅ **Early return** on first error
- ✅ **Custom validation** logic in converter functions

## Example 6: Receiver Methods

**Scenario:** Generate methods on existing types

```go
package domain

type User struct {
    ID   int
    Name string
    Email string
}

// Storage types in different package
import "myapp/storage"

type Convergen interface {
    // Generate as receiver method on User
    // :recv u
    ToStorage(*User) *storage.User
    
    // Also works with return style
    // :recv u
    // :style arg
    FromStorage(*User, *storage.User)
}
```

**Generated code:**

```go
// Receiver method generated on User type
func (u *User) ToStorage() (dst *storage.User) {
    dst = &storage.User{}
    dst.ID = int64(u.ID)
    dst.Name = u.Name
    dst.Email = u.Email
    return
}

func (u *User) FromStorage(src *storage.User) {
    u.ID = int(src.ID)
    u.Name = src.Name
    u.Email = src.Email
}
```

**Key Points:**
- ✅ **Domain model enhancement** with receiver methods
- ✅ **Method-style API** for clean usage
- ✅ **Package boundaries** handled correctly

## Example 7: Complex Mappings

**Scenario:** Combine multiple advanced features

```go
import (
    "fmt"
    "time"
)

type User struct {
    ID        int
    FirstName string
    LastName  string
    Email     string
    Password  string
    CreatedAt time.Time
    Status    UserStatus
    Metadata  map[string]interface{}
}

type UserAPIResponse struct {
    ID          int                    `json:"id"`
    FullName    string                `json:"full_name"`
    Email       string                `json:"email"`
    Status      string                `json:"status"`
    CreatedDate string                `json:"created_date"`
    Version     string                `json:"version"`
    Metadata    map[string]string     `json:"metadata"`
}

type Convergen interface {
    // Complex conversion with multiple features
    // :typecast
    // :stringer
    // :skip Password
    // :conv combineNames FirstName FullName     # Custom converter for name combination
    // :conv formatDate CreatedAt CreatedDate    # Custom converter for date formatting
    // :literal Version "1.0"
    // :conv stringifyMetadata Metadata
    // :preprocess validateUser
    // :postprocess logConversion
    UserToAPI(*User) (*UserAPIResponse, error)
}

func stringifyMetadata(meta map[string]interface{}) map[string]string {
    result := make(map[string]string)
    for k, v := range meta {
        result[k] = fmt.Sprintf("%v", v)
    }
    return result
}

func validateUser(dst *UserAPIResponse, src *User) error {
    if src.Email == "" {
        return errors.New("email is required")
    }
    return nil
}

func logConversion(dst *UserAPIResponse, src *User) error {
    fmt.Printf("Converted user %d to API response\n", src.ID)
    return nil
}
```

**Generated code:**

```go
func UserToAPI(src *User) (dst *UserAPIResponse, err error) {
    dst = &UserAPIResponse{}
    
    // Preprocess hook
    err = validateUser(dst, src)
    if err != nil {
        return
    }
    
    // Field assignments
    dst.ID = src.ID
    dst.FullName = combineNames(src.FirstName)
    dst.Email = src.Email
    dst.Status = src.Status.String()
    dst.CreatedDate = formatDate(src.CreatedAt)
    dst.Version = "1.0"
    dst.Metadata = stringifyMetadata(src.Metadata)
    // Password is skipped
    
    // Postprocess hook
    err = logConversion(dst, src)
    if err != nil {
        return
    }
    
    return
}
```

**Key Points:**
- ✅ **Multiple features** working together
- ✅ **Pre/post processing** hooks for validation and logging
- ✅ **Complex transformations** with custom converters

## Example 8: Multiple Interfaces

**Scenario:** Organize converters by domain or purpose

```go
//go:build convergen

package converters

// User-related conversions
// :convergen
type UserConvergen interface {
    // :skip Password
    // :stringer
    UserToPublic(*domain.User) *api.PublicUser
    
    // :typecast
    UserToStorage(*domain.User) *storage.User
}

// Order-related conversions  
// :convergen
type OrderConvergen interface {
    // :typecast
    // :conv calculateTotal Items Total
    OrderToAPI(*domain.Order) *api.Order
    
    // :recv o
    ToStorage(*domain.Order) *storage.Order
}

func calculateTotal(items []*domain.OrderItem) decimal.Decimal {
    var total decimal.Decimal
    for _, item := range items {
        total = total.Add(item.Price.Mul(decimal.NewFromInt(int64(item.Quantity))))
    }
    return total
}
```

**Generated code creates multiple functions:**

```go
// User conversions
func UserToPublic(src *domain.User) (dst *api.PublicUser) { ... }
func UserToStorage(src *domain.User) (dst *storage.User) { ... }

// Order conversions
func OrderToAPI(src *domain.Order) (dst *api.Order) { ... }
func (o *domain.Order) ToStorage() (dst *storage.Order) { ... }
```

**Key Points:**
- ✅ **Domain organization** with multiple interfaces
- ✅ **Clean separation** of concerns
- ✅ **Flexible naming** with `:convergen` annotation

## Common Patterns Summary

| Pattern | Use Case | Key Annotations |
|---------|----------|-----------------|
| **Simple Matching** | Identical field structures | None (default) |
| **Security Filtering** | API responses, public data | `:skip` |
| **Field Mapping** | Different field names | `:map` |
| **Type Conversion** | Type differences | `:typecast`, `:stringer` |
| **Validation** | Data integrity | `:conv` with error return |
| **Domain Methods** | Rich domain models | `:recv` |
| **Complex Transform** | Multiple requirements | Combine multiple annotations |
| **Organization** | Large projects | Multiple interfaces |

## Next Steps

Now that you've seen basic patterns:

1. **[User Guide](../guide/index.md)** - Comprehensive annotation reference
2. **[Advanced Usage](../guide/advanced-usage.md)** - Complex scenarios and generics
3. **[Real-World Examples](../examples/real-world.md)** - Production patterns
4. **[Best Practices](../guide/best-practices.md)** - Team development guidelines

## Testing Your Examples

Create test functions to verify your conversions:

```go
func TestUserConversions(t *testing.T) {
    user := &User{
        ID:        1,
        FirstName: "John",
        LastName:  "Doe",
        Email:     "john@example.com",
        Status:    StatusActive,
        CreatedAt: time.Now(),
    }

    // Test public conversion
    publicUser := UserToPublic(user)
    assert.Equal(t, user.ID, publicUser.ID)
    assert.Equal(t, user.Email, publicUser.Email)
    // Password should not be set
    assert.Empty(t, publicUser.Password)

    // Test API conversion
    apiResponse, err := UserToAPI(user)
    require.NoError(t, err)
    assert.Equal(t, "John Doe", apiResponse.FullName)
    assert.Equal(t, "active", apiResponse.Status)
    assert.Equal(t, "1.0", apiResponse.Version)
}
```

Ready to tackle more advanced scenarios? Continue to the [User Guide](../guide/index.md)!