# Annotations Reference

Complete reference for all Convergen annotations. This guide covers syntax, usage patterns, and examples for every annotation type.

## Quick Reference Table

| Annotation | Location | Purpose | Example |
|------------|----------|---------|---------|
| `:convergen` | interface | Mark interface as converter | `// :convergen` |
| `:match <algorithm>` | interface, method | Set field matching strategy | `:match name` `:match none` |
| `:style <style>` | interface, method | Function signature style | `:style return` `:style arg` |
| `:recv <var>` | method | Generate receiver method | `:recv u` |
| `:reverse` | method | Reverse copy direction | `:reverse` |
| `:case` / `:case:off` | interface, method | Case sensitivity control | `:case` `:case:off` |
| `:getter` / `:getter:off` | interface, method | Include getter methods | `:getter` `:getter:off` |
| `:stringer` / `:stringer:off` | interface, method | Use String() methods | `:stringer` `:stringer:off` |
| `:typecast` / `:typecast:off` | interface, method | Allow type casting | `:typecast` `:typecast:off` |
| `:skip <pattern>` | method | Skip destination fields | `:skip Password` `:skip /^internal/` |
| `:map <src> <dst>` | method | Explicit field mapping | `:map ID UserID` `:map Name() FullName` |
| `:conv <func> <src> [dst]` | method | Custom converter function | `:conv encrypt Email` |
| `:literal <dst> <value>` | method | Assign literal value | `:literal CreatedAt time.Now()` |
| `:preprocess <func>` | method | Pre-conversion hook | `:preprocess validate` |
| `:postprocess <func>` | method | Post-conversion hook | `:postprocess cleanup` |

## Interface-Level Annotations

These annotations apply to the entire interface and affect all methods unless overridden.

### `:convergen`

Marks an interface as a converter definition, allowing interfaces with names other than "Convergen".

**Location:** Interface  
**Format:** `// :convergen`

**Example:**

```go
// :convergen
type UserConvergen interface {
    // :recv u
    ToStorage(*User) *storage.User
}

// :convergen
type OrderConvergen interface {
    // :recv o
    ToAPI(*Order) *api.Order
}
```

**Use Cases:**
- Multiple converter interfaces in one package
- Domain-specific naming conventions
- Avoiding name conflicts in large projects

### `:match <algorithm>`

Sets the field matching strategy for the interface.

**Location:** Interface, Method  
**Default:** `:match name`  
**Format:** `:match name` | `:match none`

**Algorithms:**

=== "name"
    
    **Automatic field matching by name and type**
    
    ```go
    // :match name
    type Convergen interface {
        Convert(*User) *UserDTO
    }
    ```
    
    Generated code matches fields automatically:
    ```go
    func Convert(src *User) (dst *UserDTO) {
        dst = &UserDTO{}
        dst.ID = src.ID      // Matched by name
        dst.Name = src.Name  // Matched by name
        return
    }
    ```

=== "none"
    
    **Only explicit mappings via `:map` and `:conv`**
    
    ```go
    // :match none
    type Convergen interface {
        // :map ID UserID
        // :map Name FullName
        Convert(*User) *UserDTO
    }
    ```
    
    Generated code only includes explicit mappings:
    ```go
    func Convert(src *User) (dst *UserDTO) {
        dst = &UserDTO{}
        dst.UserID = src.ID     // Explicit mapping
        dst.FullName = src.Name // Explicit mapping
        return
    }
    ```

### `:style <style>`

Controls the generated function signature style.

**Location:** Interface, Method  
**Default:** `:style return`  
**Format:** `:style return` | `:style arg`

**Styles:**

=== "return"
    
    **Destination as return value**
    
    ```go
    // :style return
    type Convergen interface {
        Convert(*User) *UserDTO
        ConvertWithError(*User) (*UserDTO, error)
    }
    ```
    
    Generated functions:
    ```go
    func Convert(src *User) (dst *UserDTO) {
        dst = &UserDTO{}
        // ... field assignments
        return
    }
    
    func ConvertWithError(src *User) (dst *UserDTO, err error) {
        dst = &UserDTO{}
        // ... field assignments with error handling
        return
    }
    ```

=== "arg"
    
    **Destination as parameter**
    
    ```go
    // :style arg
    type Convergen interface {
        Convert(*UserDTO, *User)
        ConvertWithError(*UserDTO, *User) error
    }
    ```
    
    Generated functions:
    ```go
    func Convert(dst *UserDTO, src *User) {
        dst.ID = src.ID
        dst.Name = src.Name
        // ... other assignments
    }
    
    func ConvertWithError(dst *UserDTO, src *User) (err error) {
        dst.ID = src.ID
        // ... assignments with error handling
        return
    }
    ```

### Case Sensitivity Control

Controls case-sensitive matching for field names and patterns.

**Location:** Interface, Method  
**Default:** `:case` (case-sensitive)  
**Format:** `:case` | `:case:off`

**Example:**

```go
// Interface-level: case-insensitive by default
// :case:off
type Convergen interface {
    // Method-level: override to case-sensitive
    // :case
    StrictConvert(*User) *UserDTO
    
    // Uses interface default (case-insensitive)
    FlexibleConvert(*User) *UserDTO
}
```

**Effects:**
- `:case:off` allows `UserID` to match `userid`, `USERID`, etc.
- Applies to `:match name`, `:getter`, and `:skip` patterns
- Does not affect `:map` and `:conv` (always case-sensitive)

### Getter Method Inclusion

Controls whether getter methods are included in field matching.

**Location:** Interface, Method  
**Default:** `:getter:off`  
**Format:** `:getter` | `:getter:off`

**Example:**

```go
type User struct {
    name string // private field
}

func (u *User) Name() string {
    return u.name
}

type UserDTO struct {
    Name string
}

// :getter
type Convergen interface {
    Convert(*User) *UserDTO
}
```

**Generated code:**

```go
func Convert(src *User) (dst *UserDTO) {
    dst = &UserDTO{}
    dst.Name = src.Name() // Uses getter method
    return
}
```

### Type Conversion Helpers

Enable automatic type conversions and string formatting.

**Location:** Interface, Method  
**Default:** `:stringer:off`, `:typecast:off`  
**Format:** `:stringer` | `:stringer:off`, `:typecast` | `:typecast:off`

**Example:**

```go
type User struct {
    ID     int
    Status UserStatus // Custom type with String() method
}

type UserStatus int
func (s UserStatus) String() string { /* ... */ }

type UserDTO struct {
    ID     int64  // Different numeric type
    Status string // String representation
}

// :typecast
// :stringer
type Convergen interface {
    Convert(*User) *UserDTO
}
```

**Generated code:**

```go
func Convert(src *User) (dst *UserDTO) {
    dst = &UserDTO{}
    dst.ID = int64(src.ID)        // Type cast applied
    dst.Status = src.Status.String() // String() method used
    return
}
```

## Method-Level Annotations

These annotations apply to specific methods and override interface-level defaults.

### `:recv <var>`

Generates the function as a method with a receiver.

**Location:** Method  
**Format:** `:recv <variable>`

**Requirements:**
- Receiver type must be in the same package as generated code
- Variable name should match existing receiver conventions

**Example:**

```go
package domain

// User type defined in same package
type User struct {
    ID   int
    Name string
}

type Convergen interface {
    // :recv u
    ToStorage(*User) *storage.User
}
```

**Generated code:**

```go
func (u *User) ToStorage() (dst *storage.User) {
    dst = &storage.User{}
    dst.ID = int64(u.ID)
    dst.Name = u.Name
    return
}
```

### `:reverse`

Reverses the copy direction in receiver methods.

**Location:** Method  
**Requirements:** Must use `:style arg`  
**Format:** `:reverse`

**Example:**

```go
type Convergen interface {
    // :style arg
    // :recv u
    // :reverse
    FromStorage(*User, *storage.User)
}
```

**Generated code:**

```go
func (u *User) FromStorage(src *storage.User) {
    u.ID = int(src.ID)    // Copy FROM src TO receiver
    u.Name = src.Name
}
```

### `:skip <pattern>`

Skips copying to specific destination fields.

**Location:** Method  
**Format:** `:skip <field-path>` | `:skip /<regex>/`

**Field Path Syntax:**
- Simple field: `:skip Name`
- Nested field: `:skip Address.Street`
- Regex pattern: `:skip /^Internal/`

**Examples:**

```go
type Convergen interface {
    // Skip single field
    // :skip Password
    UserToPublic(*User) *PublicUser
    
    // Skip multiple fields
    // :skip Password
    // :skip InternalNotes
    // :skip CreatedBy
    UserToAPI(*User) *APIUser
    
    // Skip with regex
    // :skip /^internal/
    // :skip /Secret$/
    UserToExternal(*User) *ExternalUser
}
```

### `:map <src> <dst>`

Explicitly maps source to destination fields.

**Location:** Method  
**Format:** `:map <source-expression> <destination-field>`

**Source Expression Types:**

=== "Field Path"
    
    ```go
    // :map ID UserID
    // :map Address.Street StreetAddress
    Convert(*User) *UserDTO
    ```

=== "Method Call"
    
    ```go
    // :map Name() FullName
    // :map Status.String() StatusText
    Convert(*User) *UserDTO
    ```

=== "Complex Expression"
    
    ```go
    // :map FirstName + " " + LastName FullName
    // :map CreatedAt.Format("2006-01-02") CreatedDate
    Convert(*User) *UserDTO
    ```

=== "Template Arguments"
    
    ```go
    // :map $1 ExtraField    # First additional parameter
    // :map $2 AnotherField  # Second additional parameter
    Convert(*User, string, int) *UserDTO
    ```

**Generated code:**

```go
func Convert(src *User, arg0 string, arg1 int) (dst *UserDTO) {
    dst = &UserDTO{}
    dst.UserID = src.ID
    dst.StreetAddress = src.Address.Street
    dst.FullName = src.Name()
    dst.StatusText = src.Status.String()
    dst.FullName = src.FirstName + " " + src.LastName
    dst.CreatedDate = src.CreatedAt.Format("2006-01-02")
    dst.ExtraField = arg0
    dst.AnotherField = arg1
    return
}
```

### `:conv <func> <src> [dst]`

Applies custom converter function to field transformation.

**Location:** Method  
**Format:** `:conv <function> <source-field> [destination-field]`

**Converter Function Requirements:**

=== "Simple Converter"
    
    **Signature:** `func(srcType) dstType`
    
    ```go
    // :conv hashPassword Password
    Convert(*User) *UserDTO
    
    func hashPassword(password string) string {
        // Hash the password
        return hashedPassword
    }
    ```

=== "Error-Returning Converter"
    
    **Signature:** `func(srcType) (dstType, error)`
    
    ```go
    // :conv validateEmail Email
    Convert(*User) (*UserDTO, error)
    
    func validateEmail(email string) (string, error) {
        if !isValidEmail(email) {
            return "", errors.New("invalid email")
        }
        return email, nil
    }
    ```

**Examples:**

```go
import (
    "crypto/sha256"
    "encoding/hex"
    "errors"
    "strings"
)

type Convergen interface {
    // :conv hashPassword Password
    // :conv validateEmail Email  
    // :conv formatName Name FullName
    Convert(*User) (*UserDTO, error)
}

func hashPassword(password string) string {
    hash := sha256.Sum256([]byte(password))
    return hex.EncodeToString(hash[:])
}

func validateEmail(email string) (string, error) {
    if !strings.Contains(email, "@") {
        return "", errors.New("invalid email format")
    }
    return strings.ToLower(email), nil
}

func formatName(name string) string {
    return strings.Title(strings.ToLower(name))
}
```

**Generated code:**

```go
func Convert(src *User) (dst *UserDTO, err error) {
    dst = &UserDTO{}
    dst.Password = hashPassword(src.Password)
    dst.Email, err = validateEmail(src.Email)
    if err != nil {
        return
    }
    dst.FullName = formatName(src.Name)
    return
}
```

### `:literal <dst> <value>`

Assigns literal values to destination fields.

**Location:** Method  
**Format:** `:literal <destination-field> <expression>`

**Example:**

```go
import "time"

type Convergen interface {
    // :literal CreatedAt time.Now()
    // :literal Version "1.0"
    // :literal IsActive true
    // :literal Count len(src.Items)
    Convert(*User) *UserDTO
}
```

**Generated code:**

```go
func Convert(src *User) (dst *UserDTO) {
    dst = &UserDTO{}
    dst.ID = src.ID
    dst.Name = src.Name
    dst.CreatedAt = time.Now()
    dst.Version = "1.0"
    dst.IsActive = true
    dst.Count = len(src.Items)
    return
}
```

### Processing Hooks

Execute custom functions before or after the main conversion logic.

**Location:** Method  
**Format:** `:preprocess <function>` | `:postprocess <function>`

**Function Signatures:**

=== "Without Error"
    
    ```go
    func preprocess(dst *DstType, src *SrcType) {}
    func postprocess(dst *DstType, src *SrcType) *DstType {}
    ```

=== "With Error"
    
    ```go
    func preprocess(dst *DstType, src *SrcType) error {}
    func postprocess(dst *DstType, src *SrcType) error {}
    ```

=== "With Additional Arguments"
    
    ```go
    func preprocess(dst *DstType, src *SrcType, arg0 int, arg1 string) error {}
    func postprocess(dst *DstType, src *SrcType, arg0 int, arg1 string) error {}
    ```

**Example:**

```go
type Convergen interface {
    // :preprocess validateInput
    // :postprocess auditConversion
    Convert(*User, string) (*UserDTO, error)
}

func validateInput(dst *UserDTO, src *User, context string) error {
    if src.Email == "" {
        return errors.New("email is required")
    }
    return nil
}

func auditConversion(dst *UserDTO, src *User, context string) error {
    log.Printf("Converted user %d in context %s", src.ID, context)
    return nil
}
```

**Generated code:**

```go
func Convert(src *User, arg0 string) (dst *UserDTO, err error) {
    dst = &UserDTO{}
    
    // Preprocess hook
    err = validateInput(dst, src, arg0)
    if err != nil {
        return
    }
    
    // Main conversion logic
    dst.ID = src.ID
    dst.Email = src.Email
    // ... other field assignments
    
    // Postprocess hook
    err = auditConversion(dst, src, arg0)
    if err != nil {
        return
    }
    
    return
}
```

## Advanced Patterns

### Combining Annotations

Multiple annotations work together to create sophisticated conversion logic:

```go
type Convergen interface {
    // Complex conversion with multiple annotations
    // :typecast                                    # Enable type casting
    // :stringer                                   # Use String() methods
    // :skip Password                              # Skip sensitive data
    // :skip /^internal/                           # Skip internal fields
    // :map FirstName + " " + LastName FullName    # Combine fields
    // :conv hashEmail Email HashedEmail           # Custom conversion
    // :literal CreatedAt time.Now()               # Set creation time
    // :preprocess validateUser                    # Validate before conversion
    // :postprocess logConversion                  # Log after conversion
    ComplexConvert(*User) (*UserDTO, error)
}
```

### Conditional Logic

Use custom converters for conditional logic:

```go
// :conv setStatus Status
Convert(*User) *UserDTO

func setStatus(user *User) string {
    if user.IsActive && user.EmailVerified {
        return "active"
    } else if user.IsActive {
        return "pending"
    }
    return "inactive"
}
```

### Collection Handling

Handle slices and maps with custom converters:

```go
// :conv convertTags Tags
// :conv convertMetadata Metadata
Convert(*User) *UserDTO

func convertTags(tags []string) []string {
    result := make([]string, len(tags))
    for i, tag := range tags {
        result[i] = strings.ToLower(tag)
    }
    return result
}

func convertMetadata(meta map[string]interface{}) map[string]string {
    result := make(map[string]string)
    for k, v := range meta {
        result[k] = fmt.Sprintf("%v", v)
    }
    return result
}
```

## Best Practices

### Annotation Organization

```go
type Convergen interface {
    // Group related annotations together
    // Field matching configuration
    // :match name
    // :case:off
    // :getter
    //
    // Type conversion helpers  
    // :typecast
    // :stringer
    //
    // Field-specific rules
    // :skip Password
    // :map ID UserID
    // :conv hashEmail Email
    //
    // Processing hooks
    // :preprocess validate
    // :postprocess audit
    Convert(*User) (*UserDTO, error)
}
```

### Error Handling Patterns

```go
type Convergen interface {
    // Simple conversion (no errors expected)
    // :typecast
    SimpleConvert(*User) *UserDTO
    
    // Validation conversion (errors possible)
    // :conv validateEmail Email
    // :conv checkAge Age
    ValidatedConvert(*User) (*UserDTO, error)
    
    // Complex conversion with hooks
    // :preprocess authorize
    // :conv encryptData Data
    // :postprocess auditLog
    SecureConvert(*User, *AuthContext) (*UserDTO, error)
}
```

### Performance Considerations

```go
type Convergen interface {
    // Pre-allocate slices for better performance
    // :literal Items make([]*Item, 0, len(src.Items))
    // :conv convertItems Items
    OptimizedConvert(*Order) *OrderDTO
}

func convertItems(items []*OrderItem) []*ItemDTO {
    // Efficient slice conversion
    result := make([]*ItemDTO, len(items))
    for i, item := range items {
        result[i] = ItemToDTO(item)
    }
    return result
}
```

## Next Steps

Now that you understand all available annotations:

1. **[Advanced Usage](advanced-usage.md)** - Complex scenarios and patterns
2. **[Best Practices](best-practices.md)** - Team development guidelines  
3. **[Examples](../examples/real-world.md)** - Real-world usage patterns
4. **[Performance](performance.md)** - Optimization techniques