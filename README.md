# Convergen

[![Go Reference](https://pkg.go.dev/badge/github.com/reedom/convergen.svg)](https://pkg.go.dev/github.com/reedom/convergen) 
[![Go Report Card](https://goreportcard.com/badge/github.com/reedom/convergen)](https://goreportcard.com/report/github.com/reedom/convergen) 
![Coverage](https://img.shields.io/badge/Coverage-67.4%25-yellow)
[![Documentation](https://img.shields.io/badge/docs-convergen-blue)](https://reedom.github.io/convergen/)

Convergen is a **high-performance** code generator that creates type-to-type copy functions from annotated interfaces. Write an interface with annotations, and Convergen generates efficient conversion code that handles field mapping, type casting, and custom transformations.

**📚 [Complete Documentation](https://reedom.github.io/convergen/) • [Quick Start](https://reedom.github.io/convergen/getting-started/quick-start/) • [Examples](https://reedom.github.io/convergen/examples/) • [API Reference](https://reedom.github.io/convergen/api/)**

## ⚡ Key Features

- **🚀 High Performance**: 40-70% faster parsing with concurrent processing (v9)
- **🎯 Type-Safe**: Leverages Go's type system for safe conversions
- **🔧 Flexible**: Supports field mapping, type casting, custom converters
- **📦 Zero Runtime Dependencies**: Generated code has no external dependencies
- **🏗️ Production Ready**: Enterprise reliability with comprehensive error handling

## Installation

### As Go Generator (Recommended)
```bash
go get -u github.com/reedom/convergen@latest
```

### As CLI Tool
```bash
go install github.com/reedom/convergen@latest
```

## Quick Start

1. **Create a converter interface**:

```go
//go:build convergen

package sample

//go:generate go run github.com/reedom/convergen@latest
type Convergen interface {
    // :typecast :stringer
    DomainToStorage(*domain.User) *storage.User
}
```

2. **Generate the code**:
```bash
go generate
```

3. **Use the generated function**:
```go
domainUser := &domain.User{ID: 1, Name: "John"}
storageUser := DomainToStorage(domainUser)
```

### Example Generated Code

From this input:
```go
type Convergen interface {
    // :typecast
    // :map Created.UnixMilli() Created  
    DomainToStorage(*domain.User) *storage.User
}
```

Convergen generates:
```go
func DomainToStorage(src *domain.User) (dst *storage.User) {
    dst = &storage.User{}
    dst.ID = int64(src.ID)           // typecast
    dst.Name = src.Name              // direct copy
    dst.Created = src.Created.UnixMilli() // method mapping
    return
}
```

## Common Annotations

| Annotation | Purpose | Example |
|------------|---------|---------|
| `:typecast` | Allow type casting | `int` → `int64` |
| `:stringer` | Use String() methods | `Status` → `string` |
| `:map <src> <dst>` | Map fields explicitly | `:map ID UserID` |
| `:skip <field>` | Skip destination fields | `:skip Password` |
| `:conv <func> <src>` | Custom converter function | `:conv encrypt Email` |

## Documentation

For comprehensive documentation including all annotations, advanced examples, and best practices:

**📚 [https://reedom.github.io/convergen/](https://reedom.github.io/convergen/)**

## Contributing

Contributions are welcome! You can help by:

- 🐛 [Reporting bugs](https://github.com/reedom/convergen/issues)
- 💡 [Suggesting features](https://github.com/reedom/convergen/issues)
- 🔧 [Contributing code](https://github.com/reedom/convergen/pulls)
- 📚 [Improving documentation](https://github.com/reedom/convergen/pulls)
- ⭐ [Giving the project a star](https://github.com/reedom/convergen)
