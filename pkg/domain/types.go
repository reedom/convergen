package domain

import (
	"errors"
	"fmt"
	"reflect"
)

// Static errors for err113 compliance.
var (
	ErrStructNameEmpty    = errors.New("struct name cannot be empty")
	ErrDuplicateFieldName = errors.New("duplicate field name")
	ErrFieldHasNilType    = errors.New("field has nil type")
)

// TypeKind represents the fundamental kind of a type.
type TypeKind int

const (
	// KindBasic represents basic types like int, string, bool.
	KindBasic TypeKind = iota
	// KindStruct represents struct types.
	KindStruct
	// KindSlice represents slice types.
	KindSlice
	// KindMap represents map types.
	KindMap
	// KindInterface represents interface types.
	KindInterface
	// KindPointer represents pointer types.
	KindPointer
	// KindGeneric represents generic types.
	KindGeneric
	// KindNamed represents named types.
	KindNamed
	// KindFunction represents function types.
	KindFunction

	// TypeKindInterface is an alias for KindInterface for compatibility.
	TypeKindInterface = KindInterface
)

func (k TypeKind) String() string {
	switch k {
	case KindBasic:
		return "basic"
	case KindStruct:
		return "struct"
	case KindSlice:
		return "slice"
	case KindMap:
		return "map"
	case KindInterface:
		return "interface"
	case KindPointer:
		return "pointer"
	case KindGeneric:
		return "generic"
	case KindNamed:
		return "named"
	case KindFunction:
		return "function"
	default:
		return "unknown"
	}
}

// Type represents a Go type with full information including generics.
type Type interface {
	// Basic type information
	Name() string
	Kind() TypeKind
	String() string

	// Generic type support
	Generic() bool
	TypeParams() []TypeParam

	// Type relationships
	Underlying() Type
	AssignableTo(other Type) bool
	Implements(iface Type) bool
	Comparable() bool

	// Package information
	Package() string
	ImportPath() string
}

// TypeParam represents a generic type parameter.
type TypeParam struct {
	Name       string `json:"name"`
	Constraint Type   `json:"constraint"`
	Index      int    `json:"index"`
}

// BasicType represents primitive types (int, string, bool, etc.)
type BasicType struct {
	name string
	kind reflect.Kind
	pkg  string
}

// NewBasicType creates a new basic type with the given name and kind.
func NewBasicType(name string, kind reflect.Kind) *BasicType {
	return &BasicType{
		name: name,
		kind: kind,
		pkg:  "",
	}
}

// Name returns the name of the basic type.
func (t *BasicType) Name() string { return t.name }

// Kind returns the type kind, always KindBasic for basic types.
func (t *BasicType) Kind() TypeKind { return KindBasic }

// String returns the string representation of the basic type.
func (t *BasicType) String() string { return t.name }

// Generic returns false as basic types are not generic.
func (t *BasicType) Generic() bool { return false }

// TypeParams returns nil as basic types have no type parameters.
func (t *BasicType) TypeParams() []TypeParam { return nil }

// Underlying returns the type itself as basic types are their own underlying type.
func (t *BasicType) Underlying() Type { return t }

// Package returns the package name for the basic type.
func (t *BasicType) Package() string { return t.pkg }

// ImportPath returns the import path, empty for basic types.
func (t *BasicType) ImportPath() string { return "" }

// AssignableTo checks if this basic type is assignable to another type.
func (t *BasicType) AssignableTo(other Type) bool {
	if other == nil {
		return false
	}
	// Basic types are assignable if they're identical
	if otherBasic, ok := other.(*BasicType); ok {
		return t.name == otherBasic.name && t.kind == otherBasic.kind
	}

	return false
}

// Implements checks if this basic type implements an interface.
func (t *BasicType) Implements(iface Type) bool {
	// Basic types don't implement interfaces directly
	return false
}

// Comparable returns true if the basic type is comparable.
func (t *BasicType) Comparable() bool {
	// Most basic types are comparable except functions, maps, slices
	switch t.kind {
	case reflect.Func, reflect.Map, reflect.Slice:
		return false
	default:
		return true
	}
}

// StructType represents struct types with ordered fields.
type StructType struct {
	name       string
	fields     []Field
	typeParams []TypeParam
	pkg        string
	importPath string
}

// NewStructType creates a new struct type with the given name, fields, and package.
func NewStructType(name string, fields []Field, pkg string) *StructType {
	return &StructType{
		name:       name,
		fields:     append([]Field(nil), fields...), // defensive copy
		typeParams: nil,
		pkg:        pkg,
		importPath: pkg,
	}
}

// Name returns the name of the struct type.
func (t *StructType) Name() string { return t.name }

// Kind returns the type kind, always KindStruct for struct types.
func (t *StructType) Kind() TypeKind { return KindStruct }

// String returns the string representation of the struct type.
func (t *StructType) String() string { return t.name }

// Generic returns true if the struct type has type parameters.
func (t *StructType) Generic() bool { return len(t.typeParams) > 0 }

// TypeParams returns a copy of the type parameters.
func (t *StructType) TypeParams() []TypeParam { return append([]TypeParam(nil), t.typeParams...) }

// Underlying returns the type itself as struct types are their own underlying type.
func (t *StructType) Underlying() Type { return t }

// Package returns the package name for the struct type.
func (t *StructType) Package() string { return t.pkg }

// ImportPath returns the import path for the struct type.
func (t *StructType) ImportPath() string { return t.importPath }

// Fields returns a defensive copy of the fields.
func (t *StructType) Fields() []Field {
	return append([]Field(nil), t.fields...)
}

// FieldByName finds a field by name.
func (t *StructType) FieldByName(name string) (Field, bool) {
	for _, field := range t.fields {
		if field.Name == name {
			return field, true
		}
	}

	return Field{}, false
}

// AssignableTo checks if this struct type is assignable to another type.
func (t *StructType) AssignableTo(other Type) bool {
	if other == nil {
		return false
	}
	// Structs are assignable if they're identical
	if otherStruct, ok := other.(*StructType); ok {
		return t.name == otherStruct.name && t.pkg == otherStruct.pkg
	}

	return false
}

// Implements checks if this struct type implements an interface.
func (t *StructType) Implements(iface Type) bool {
	// TODO: Implement interface satisfaction checking
	return false
}

// Comparable returns true if the struct type is comparable.
func (t *StructType) Comparable() bool {
	// Structs are comparable if all their fields are comparable
	for _, field := range t.fields {
		if !field.Type.Comparable() {
			return false
		}
	}

	return true
}

// SliceType represents slice types.
type SliceType struct {
	elem       Type
	pkg        string
	importPath string
}

// NewSliceType creates a new slice type with the given element type and package.
func NewSliceType(elem Type, pkg string) *SliceType {
	return &SliceType{
		elem:       elem,
		pkg:        pkg,
		importPath: pkg,
	}
}

// Name returns the name of the slice type.
func (t *SliceType) Name() string { return "[]" + t.elem.Name() }

// Kind returns the type kind, always KindSlice for slice types.
func (t *SliceType) Kind() TypeKind { return KindSlice }

// String returns the string representation of the slice type.
func (t *SliceType) String() string { return "[]" + t.elem.String() }

// Generic returns true if the element type is generic.
func (t *SliceType) Generic() bool { return t.elem.Generic() }

// TypeParams returns the type parameters from the element type.
func (t *SliceType) TypeParams() []TypeParam { return t.elem.TypeParams() }

// Underlying returns the type itself as slice types are their own underlying type.
func (t *SliceType) Underlying() Type { return t }

// Package returns the package name for the slice type.
func (t *SliceType) Package() string { return t.pkg }

// ImportPath returns the import path for the slice type.
func (t *SliceType) ImportPath() string { return t.importPath }

// Elem returns the element type of the slice.
func (t *SliceType) Elem() Type { return t.elem }

// AssignableTo checks if this slice type is assignable to another type.
func (t *SliceType) AssignableTo(other Type) bool {
	if other == nil {
		return false
	}

	if otherSlice, ok := other.(*SliceType); ok {
		return t.elem.AssignableTo(otherSlice.elem)
	}

	return false
}

// Implements checks if this slice type implements an interface.
func (t *SliceType) Implements(iface Type) bool {
	return false
}

// Comparable returns false as slice types are not comparable.
func (t *SliceType) Comparable() bool {
	return false // Slices are not comparable
}

// PointerType represents pointer types.
type PointerType struct {
	elem       Type
	pkg        string
	importPath string
}

// NewPointerType creates a new pointer type with the given element type and package.
func NewPointerType(elem Type, pkg string) *PointerType {
	return &PointerType{
		elem:       elem,
		pkg:        pkg,
		importPath: pkg,
	}
}

// Name returns the name of the pointer type.
func (t *PointerType) Name() string { return "*" + t.elem.Name() }

// Kind returns the type kind, always KindPointer for pointer types.
func (t *PointerType) Kind() TypeKind { return KindPointer }

// String returns the string representation of the pointer type.
func (t *PointerType) String() string { return "*" + t.elem.String() }

// Generic returns true if the element type is generic.
func (t *PointerType) Generic() bool { return t.elem.Generic() }

// TypeParams returns the type parameters from the element type.
func (t *PointerType) TypeParams() []TypeParam { return t.elem.TypeParams() }

// Underlying returns the type itself as pointer types are their own underlying type.
func (t *PointerType) Underlying() Type { return t }

// Package returns the package name for the pointer type.
func (t *PointerType) Package() string { return t.pkg }

// ImportPath returns the import path for the pointer type.
func (t *PointerType) ImportPath() string { return t.importPath }

// Elem returns the element type that this pointer points to.
func (t *PointerType) Elem() Type { return t.elem }

// AssignableTo checks if this pointer type is assignable to another type.
func (t *PointerType) AssignableTo(other Type) bool {
	if other == nil {
		return false
	}

	if otherPtr, ok := other.(*PointerType); ok {
		return t.elem.AssignableTo(otherPtr.elem)
	}

	return false
}

// Implements checks if this pointer type implements an interface.
func (t *PointerType) Implements(iface Type) bool {
	// Pointer types can implement interfaces through their element type
	return t.elem.Implements(iface)
}

// Comparable returns true as pointer types are always comparable.
func (t *PointerType) Comparable() bool {
	return true // Pointers are always comparable
}

// GenericType represents generic type parameters.
type GenericType struct {
	name       string
	constraint Type
	index      int
	pkg        string
	importPath string
}

// NewGenericType creates a new generic type with the given name, constraint, index, and package.
func NewGenericType(name string, constraint Type, index int, pkg string) *GenericType {
	return &GenericType{
		name:       name,
		constraint: constraint,
		index:      index,
		pkg:        pkg,
		importPath: pkg,
	}
}

// Name returns the name of the generic type.
func (t *GenericType) Name() string { return t.name }

// Kind returns the type kind, always KindGeneric for generic types.
func (t *GenericType) Kind() TypeKind { return KindGeneric }

// String returns the string representation of the generic type.
func (t *GenericType) String() string { return t.name }

// Generic returns true as this is a generic type.
func (t *GenericType) Generic() bool { return true }

// TypeParams returns the type parameters for this generic type.
func (t *GenericType) TypeParams() []TypeParam {
	return []TypeParam{{Name: t.name, Constraint: t.constraint, Index: t.index}}
}

// Underlying returns the constraint type as the underlying type.
func (t *GenericType) Underlying() Type { return t.constraint }

// Package returns the package name for the generic type.
func (t *GenericType) Package() string { return t.pkg }

// ImportPath returns the import path for the generic type.
func (t *GenericType) ImportPath() string { return t.importPath }

// Constraint returns the constraint type for this generic type.
func (t *GenericType) Constraint() Type { return t.constraint }

// Index returns the index of this type parameter.
func (t *GenericType) Index() int { return t.index }

// AssignableTo checks if this generic type is assignable to another type.
func (t *GenericType) AssignableTo(other Type) bool {
	if other == nil {
		return false
	}
	// Generic types are assignable based on their constraints
	return t.constraint.AssignableTo(other)
}

// Implements checks if this generic type implements an interface.
func (t *GenericType) Implements(iface Type) bool {
	return t.constraint.Implements(iface)
}

// Comparable returns true if the constraint type is comparable.
func (t *GenericType) Comparable() bool {
	return t.constraint.Comparable()
}

// TypeBuilder helps build complex types with validation.
type TypeBuilder struct {
	cache map[string]Type
}

// NewTypeBuilder creates a new type builder with caching.
func NewTypeBuilder() *TypeBuilder {
	return &TypeBuilder{
		cache: make(map[string]Type),
	}
}

// NewNamedType creates a new named type with the given parameters.
func NewNamedType(name string, underlying Type, typeParams []TypeParam) Type {
	return &BasicType{
		name: name,
		kind: reflect.Struct, // Default for named types
		pkg:  "",
	}
}

// NewArrayType creates a new array type with the given element type and length.
func NewArrayType(elem Type, length int) Type {
	return &SliceType{
		elem: elem,
		pkg:  "",
	}
}

// NewMapType creates a new map type with the given key and value types.
func NewMapType(key, value Type) Type {
	return &mapType{
		name:  "map[" + key.Name() + "]" + value.Name(),
		key:   key,
		value: value,
	}
}

// mapType is a simple wrapper that returns KindMap.
type mapType struct {
	name  string
	key   Type
	value Type
}

func (t *mapType) Name() string                 { return t.name }
func (t *mapType) Kind() TypeKind               { return KindMap }
func (t *mapType) String() string               { return t.name }
func (t *mapType) Generic() bool                { return false }
func (t *mapType) TypeParams() []TypeParam      { return nil }
func (t *mapType) Underlying() Type             { return t }
func (t *mapType) Package() string              { return "" }
func (t *mapType) ImportPath() string           { return "" }
func (t *mapType) AssignableTo(other Type) bool { return false }
func (t *mapType) Implements(iface Type) bool   { return false }
func (t *mapType) Comparable() bool             { return false }

// NewInterfaceType creates a new interface type with the given methods.
func NewInterfaceType(methods []*Method) Type {
	return &BasicType{
		name: "interface{}",
		kind: reflect.Interface,
		pkg:  "",
	}
}

// NewChannelType creates a new channel type with the given element type and direction.
func NewChannelType(elem Type, direction ChannelDirection) Type {
	return &BasicType{
		name: "chan " + elem.Name(),
		kind: reflect.Chan,
		pkg:  "",
	}
}

// NewFunctionType creates a new function type with the given parameters, returns, and variadic flag.
func NewFunctionType(params, returns []Type, variadic bool) Type {
	return &functionType{
		name:     "func",
		params:   params,
		returns:  returns,
		variadic: variadic,
	}
}

// functionType is a simple wrapper that returns KindFunction.
type functionType struct {
	name     string
	params   []Type
	returns  []Type
	variadic bool
}

func (t *functionType) Name() string                 { return t.name }
func (t *functionType) Kind() TypeKind               { return KindFunction }
func (t *functionType) String() string               { return t.name }
func (t *functionType) Generic() bool                { return false }
func (t *functionType) TypeParams() []TypeParam      { return nil }
func (t *functionType) Underlying() Type             { return t }
func (t *functionType) Package() string              { return "" }
func (t *functionType) ImportPath() string           { return "" }
func (t *functionType) AssignableTo(other Type) bool { return false }
func (t *functionType) Implements(iface Type) bool   { return false }
func (t *functionType) Comparable() bool             { return false }

// NewTypeParameterType creates a new type parameter with the given name and constraint.
func NewTypeParameterType(name string, constraint Type) Type {
	return &GenericType{
		name:       name,
		constraint: constraint,
		index:      0,
		pkg:        "",
	}
}

// BuildStruct creates a validated struct type.
func (b *TypeBuilder) BuildStruct(name, pkg string, fields []Field) (*StructType, error) {
	if name == "" {
		return nil, ErrStructNameEmpty
	}

	// Validate field names are unique
	fieldNames := make(map[string]bool)
	for _, field := range fields {
		if fieldNames[field.Name] {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateFieldName, field.Name)
		}

		fieldNames[field.Name] = true

		if field.Type == nil {
			return nil, fmt.Errorf("%w: %s", ErrFieldHasNilType, field.Name)
		}
	}

	structType := NewStructType(name, fields, pkg)

	// Cache the type
	key := fmt.Sprintf("%s.%s", pkg, name)
	b.cache[key] = structType

	return structType, nil
}

// GetCachedType retrieves a cached type.
func (b *TypeBuilder) GetCachedType(pkg, name string) (Type, bool) {
	key := fmt.Sprintf("%s.%s", pkg, name)
	typ, ok := b.cache[key]

	return typ, ok
}

// Common basic types for convenience.
var (
	StringType  = NewBasicType("string", reflect.String)
	IntType     = NewBasicType("int", reflect.Int)
	Int8Type    = NewBasicType("int8", reflect.Int8)
	Int16Type   = NewBasicType("int16", reflect.Int16)
	Int32Type   = NewBasicType("int32", reflect.Int32)
	Int64Type   = NewBasicType("int64", reflect.Int64)
	UintType    = NewBasicType("uint", reflect.Uint)
	Uint8Type   = NewBasicType("uint8", reflect.Uint8)
	Uint16Type  = NewBasicType("uint16", reflect.Uint16)
	Uint32Type  = NewBasicType("uint32", reflect.Uint32)
	Uint64Type  = NewBasicType("uint64", reflect.Uint64)
	Float32Type = NewBasicType("float32", reflect.Float32)
	Float64Type = NewBasicType("float64", reflect.Float64)
	BoolType    = NewBasicType("bool", reflect.Bool)
	ByteType    = NewBasicType("byte", reflect.Uint8)
	RuneType    = NewBasicType("rune", reflect.Int32)
)
