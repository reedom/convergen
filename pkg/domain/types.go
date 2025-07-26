package domain

import (
	"fmt"
	"reflect"
)

// TypeKind represents the fundamental kind of a type
type TypeKind int

const (
	KindBasic TypeKind = iota
	KindStruct
	KindSlice
	KindMap
	KindInterface
	KindPointer
	KindGeneric
	KindNamed
	KindFunction

	// Aliases for compatibility
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

// Type represents a Go type with full information including generics
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

// TypeParam represents a generic type parameter
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

func NewBasicType(name string, kind reflect.Kind) *BasicType {
	return &BasicType{
		name: name,
		kind: kind,
		pkg:  "",
	}
}

func (t *BasicType) Name() string            { return t.name }
func (t *BasicType) Kind() TypeKind          { return KindBasic }
func (t *BasicType) String() string          { return t.name }
func (t *BasicType) Generic() bool           { return false }
func (t *BasicType) TypeParams() []TypeParam { return nil }
func (t *BasicType) Underlying() Type        { return t }
func (t *BasicType) Package() string         { return t.pkg }
func (t *BasicType) ImportPath() string      { return "" }

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

func (t *BasicType) Implements(iface Type) bool {
	// Basic types don't implement interfaces directly
	return false
}

func (t *BasicType) Comparable() bool {
	// Most basic types are comparable except functions, maps, slices
	switch t.kind {
	case reflect.Func, reflect.Map, reflect.Slice:
		return false
	default:
		return true
	}
}

// StructType represents struct types with ordered fields
type StructType struct {
	name       string
	fields     []Field
	typeParams []TypeParam
	pkg        string
	importPath string
}

func NewStructType(name string, fields []Field, pkg string) *StructType {
	return &StructType{
		name:       name,
		fields:     append([]Field(nil), fields...), // defensive copy
		typeParams: nil,
		pkg:        pkg,
		importPath: pkg,
	}
}

func (t *StructType) Name() string            { return t.name }
func (t *StructType) Kind() TypeKind          { return KindStruct }
func (t *StructType) String() string          { return t.name }
func (t *StructType) Generic() bool           { return len(t.typeParams) > 0 }
func (t *StructType) TypeParams() []TypeParam { return append([]TypeParam(nil), t.typeParams...) }
func (t *StructType) Underlying() Type        { return t }
func (t *StructType) Package() string         { return t.pkg }
func (t *StructType) ImportPath() string      { return t.importPath }

// Fields returns a defensive copy of the fields
func (t *StructType) Fields() []Field {
	return append([]Field(nil), t.fields...)
}

// FieldByName finds a field by name
func (t *StructType) FieldByName(name string) (Field, bool) {
	for _, field := range t.fields {
		if field.Name == name {
			return field, true
		}
	}
	return Field{}, false
}

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

func (t *StructType) Implements(iface Type) bool {
	// TODO: Implement interface satisfaction checking
	return false
}

func (t *StructType) Comparable() bool {
	// Structs are comparable if all their fields are comparable
	for _, field := range t.fields {
		if !field.Type.Comparable() {
			return false
		}
	}
	return true
}

// SliceType represents slice types
type SliceType struct {
	elem       Type
	pkg        string
	importPath string
}

func NewSliceType(elem Type, pkg string) *SliceType {
	return &SliceType{
		elem:       elem,
		pkg:        pkg,
		importPath: pkg,
	}
}

func (t *SliceType) Name() string            { return "[]" + t.elem.Name() }
func (t *SliceType) Kind() TypeKind          { return KindSlice }
func (t *SliceType) String() string          { return "[]" + t.elem.String() }
func (t *SliceType) Generic() bool           { return t.elem.Generic() }
func (t *SliceType) TypeParams() []TypeParam { return t.elem.TypeParams() }
func (t *SliceType) Underlying() Type        { return t }
func (t *SliceType) Package() string         { return t.pkg }
func (t *SliceType) ImportPath() string      { return t.importPath }
func (t *SliceType) Elem() Type              { return t.elem }

func (t *SliceType) AssignableTo(other Type) bool {
	if other == nil {
		return false
	}
	if otherSlice, ok := other.(*SliceType); ok {
		return t.elem.AssignableTo(otherSlice.elem)
	}
	return false
}

func (t *SliceType) Implements(iface Type) bool {
	return false
}

func (t *SliceType) Comparable() bool {
	return false // Slices are not comparable
}

// PointerType represents pointer types
type PointerType struct {
	elem       Type
	pkg        string
	importPath string
}

func NewPointerType(elem Type, pkg string) *PointerType {
	return &PointerType{
		elem:       elem,
		pkg:        pkg,
		importPath: pkg,
	}
}

func (t *PointerType) Name() string            { return "*" + t.elem.Name() }
func (t *PointerType) Kind() TypeKind          { return KindPointer }
func (t *PointerType) String() string          { return "*" + t.elem.String() }
func (t *PointerType) Generic() bool           { return t.elem.Generic() }
func (t *PointerType) TypeParams() []TypeParam { return t.elem.TypeParams() }
func (t *PointerType) Underlying() Type        { return t }
func (t *PointerType) Package() string         { return t.pkg }
func (t *PointerType) ImportPath() string      { return t.importPath }
func (t *PointerType) Elem() Type              { return t.elem }

func (t *PointerType) AssignableTo(other Type) bool {
	if other == nil {
		return false
	}
	if otherPtr, ok := other.(*PointerType); ok {
		return t.elem.AssignableTo(otherPtr.elem)
	}
	return false
}

func (t *PointerType) Implements(iface Type) bool {
	// Pointer types can implement interfaces through their element type
	return t.elem.Implements(iface)
}

func (t *PointerType) Comparable() bool {
	return true // Pointers are always comparable
}

// GenericType represents generic type parameters
type GenericType struct {
	name       string
	constraint Type
	index      int
	pkg        string
	importPath string
}

func NewGenericType(name string, constraint Type, index int, pkg string) *GenericType {
	return &GenericType{
		name:       name,
		constraint: constraint,
		index:      index,
		pkg:        pkg,
		importPath: pkg,
	}
}

func (t *GenericType) Name() string   { return t.name }
func (t *GenericType) Kind() TypeKind { return KindGeneric }
func (t *GenericType) String() string { return t.name }
func (t *GenericType) Generic() bool  { return true }
func (t *GenericType) TypeParams() []TypeParam {
	return []TypeParam{{Name: t.name, Constraint: t.constraint, Index: t.index}}
}
func (t *GenericType) Underlying() Type   { return t.constraint }
func (t *GenericType) Package() string    { return t.pkg }
func (t *GenericType) ImportPath() string { return t.importPath }
func (t *GenericType) Constraint() Type   { return t.constraint }
func (t *GenericType) Index() int         { return t.index }

func (t *GenericType) AssignableTo(other Type) bool {
	if other == nil {
		return false
	}
	// Generic types are assignable based on their constraints
	return t.constraint.AssignableTo(other)
}

func (t *GenericType) Implements(iface Type) bool {
	return t.constraint.Implements(iface)
}

func (t *GenericType) Comparable() bool {
	return t.constraint.Comparable()
}

// TypeBuilder helps build complex types with validation
type TypeBuilder struct {
	cache map[string]Type
}

func NewTypeBuilder() *TypeBuilder {
	return &TypeBuilder{
		cache: make(map[string]Type),
	}
}

// Additional constructors needed by the parser
func NewNamedType(name string, underlying Type, typeParams []TypeParam) Type {
	return &BasicType{
		name: name,
		kind: reflect.Struct, // Default for named types
		pkg:  "",
	}
}

func NewArrayType(elem Type, length int) Type {
	return &SliceType{
		elem: elem,
		pkg:  "",
	}
}

func NewMapType(key, value Type) Type {
	return &mapType{
		name:  "map[" + key.Name() + "]" + value.Name(),
		key:   key,
		value: value,
	}
}

// mapType is a simple wrapper that returns KindMap
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

func NewInterfaceType(methods []*Method) Type {
	return &BasicType{
		name: "interface{}",
		kind: reflect.Interface,
		pkg:  "",
	}
}

func NewChannelType(elem Type, direction ChannelDirection) Type {
	return &BasicType{
		name: "chan " + elem.Name(),
		kind: reflect.Chan,
		pkg:  "",
	}
}

func NewFunctionType(params, returns []Type, variadic bool) Type {
	return &functionType{
		name:     "func",
		params:   params,
		returns:  returns,
		variadic: variadic,
	}
}

// functionType is a simple wrapper that returns KindFunction
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

func NewTypeParameterType(name string, constraint Type) Type {
	return &GenericType{
		name:       name,
		constraint: constraint,
		index:      0,
		pkg:        "",
	}
}

// BuildStruct creates a validated struct type
func (b *TypeBuilder) BuildStruct(name, pkg string, fields []Field) (*StructType, error) {
	if name == "" {
		return nil, fmt.Errorf("struct name cannot be empty")
	}

	// Validate field names are unique
	fieldNames := make(map[string]bool)
	for _, field := range fields {
		if fieldNames[field.Name] {
			return nil, fmt.Errorf("duplicate field name: %s", field.Name)
		}
		fieldNames[field.Name] = true

		if field.Type == nil {
			return nil, fmt.Errorf("field %s has nil type", field.Name)
		}
	}

	structType := NewStructType(name, fields, pkg)

	// Cache the type
	key := fmt.Sprintf("%s.%s", pkg, name)
	b.cache[key] = structType

	return structType, nil
}

// GetCachedType retrieves a cached type
func (b *TypeBuilder) GetCachedType(pkg, name string) (Type, bool) {
	key := fmt.Sprintf("%s.%s", pkg, name)
	typ, ok := b.cache[key]
	return typ, ok
}

// Common basic types for convenience
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
