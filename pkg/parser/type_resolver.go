package parser

import (
	"context"
	"fmt"
	"go/types"
	"reflect"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/reedom/convergen/v8/pkg/domain"
)

// TypeResolver provides concurrent type resolution with caching
type TypeResolver struct {
	cache       *TypeCache
	logger      *zap.Logger
	typeInfoMap sync.Map // Cache for domain.TypeInfo
}

// TypeResolverPool manages a pool of type resolvers for concurrent processing
type TypeResolverPool struct {
	resolvers []*TypeResolver
	current   int
	mutex     sync.Mutex
	logger    *zap.Logger
	closed    bool
}

// NewTypeResolver creates a new type resolver
func NewTypeResolver(cache *TypeCache, logger *zap.Logger) *TypeResolver {
	return &TypeResolver{
		cache:  cache,
		logger: logger,
	}
}

// NewTypeResolverPool creates a pool of type resolvers with a shared cache
func NewTypeResolverPool(size int, cache *TypeCache, logger *zap.Logger) *TypeResolverPool {
	resolvers := make([]*TypeResolver, size)

	for i := 0; i < size; i++ {
		resolvers[i] = NewTypeResolver(cache, logger)
	}

	return &TypeResolverPool{
		resolvers: resolvers,
		logger:    logger,
	}
}

// Get returns the next available type resolver in round-robin fashion
func (p *TypeResolverPool) Get() *TypeResolver {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.closed {
		return nil
	}

	resolver := p.resolvers[p.current]
	p.current = (p.current + 1) % len(p.resolvers)
	return resolver
}

// Close closes the type resolver pool
func (p *TypeResolverPool) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.closed = true
	return nil
}

// ResolveType converts a go/types.Type to domain.Type with full generics support
func (tr *TypeResolver) ResolveType(ctx context.Context, goType types.Type) (domain.Type, error) {
	// Check cache first
	if cached := tr.cache.Get(goType.String()); cached != nil {
		return cached, nil
	}

	// Resolve type based on its underlying structure
	var domainType domain.Type
	var err error

	switch t := goType.(type) {
	case *types.Basic:
		domainType, err = tr.resolveBasicType(t)
	case *types.Named:
		domainType, err = tr.resolveNamedType(ctx, t)
	case *types.Pointer:
		domainType, err = tr.resolvePointerType(ctx, t)
	case *types.Slice:
		domainType, err = tr.resolveSliceType(ctx, t)
	case *types.Array:
		domainType, err = tr.resolveArrayType(ctx, t)
	case *types.Map:
		domainType, err = tr.resolveMapType(ctx, t)
	case *types.Struct:
		domainType, err = tr.resolveStructType(ctx, t)
	case *types.Interface:
		domainType, err = tr.resolveInterfaceType(ctx, t)
	case *types.Chan:
		domainType, err = tr.resolveChanType(ctx, t)
	case *types.Signature:
		domainType, err = tr.resolveSignatureType(ctx, t)
	case *types.TypeParam:
		domainType, err = tr.resolveTypeParam(ctx, t)
	default:
		err = fmt.Errorf("unsupported type: %T", goType)
	}

	if err != nil {
		return nil, err
	}

	// Cache the resolved type
	tr.cache.Put(goType.String(), domainType)

	return domainType, nil
}

// resolveBasicType handles basic Go types (int, string, bool, etc.)
func (tr *TypeResolver) resolveBasicType(basic *types.Basic) (domain.Type, error) {
	kind := tr.mapBasicTypeKind(basic.Kind())

	return domain.NewBasicType(basic.Name(), kind), nil
}

// resolveNamedType handles named types with generics support
func (tr *TypeResolver) resolveNamedType(ctx context.Context, named *types.Named) (domain.Type, error) {
	// Check if this is a generic type
	var typeParams []domain.TypeParam
	if named.TypeParams() != nil {
		for i := 0; i < named.TypeParams().Len(); i++ {
			param := named.TypeParams().At(i)
			constraint, err := tr.ResolveType(ctx, param.Constraint())
			if err != nil {
				return nil, fmt.Errorf("failed to resolve type parameter constraint: %w", err)
			}

			typeParams = append(typeParams, domain.TypeParam{
				Name:       param.Obj().Name(),
				Constraint: constraint,
				Index:      i,
			})
		}
	}

	// Resolve underlying type
	underlying, err := tr.ResolveType(ctx, named.Underlying())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve underlying type: %w", err)
	}

	return domain.NewNamedType(named.Obj().Name(), underlying, typeParams), nil
}

// resolvePointerType handles pointer types
func (tr *TypeResolver) resolvePointerType(ctx context.Context, ptr *types.Pointer) (domain.Type, error) {
	elem, err := tr.ResolveType(ctx, ptr.Elem())
	if err != nil {
		return nil, err
	}

	return domain.NewPointerType(elem, ""), nil
}

// resolveSliceType handles slice types
func (tr *TypeResolver) resolveSliceType(ctx context.Context, slice *types.Slice) (domain.Type, error) {
	elem, err := tr.ResolveType(ctx, slice.Elem())
	if err != nil {
		return nil, err
	}

	return domain.NewSliceType(elem, ""), nil
}

// resolveArrayType handles array types
func (tr *TypeResolver) resolveArrayType(ctx context.Context, array *types.Array) (domain.Type, error) {
	elem, err := tr.ResolveType(ctx, array.Elem())
	if err != nil {
		return nil, err
	}

	return domain.NewArrayType(elem, int(array.Len())), nil
}

// resolveMapType handles map types
func (tr *TypeResolver) resolveMapType(ctx context.Context, mapType *types.Map) (domain.Type, error) {
	key, err := tr.ResolveType(ctx, mapType.Key())
	if err != nil {
		return nil, err
	}

	value, err := tr.ResolveType(ctx, mapType.Elem())
	if err != nil {
		return nil, err
	}

	return domain.NewMapType(key, value), nil
}

// resolveStructType handles struct types
func (tr *TypeResolver) resolveStructType(ctx context.Context, structType *types.Struct) (domain.Type, error) {
	fields := make([]*domain.Field, structType.NumFields())

	// Use error group for concurrent field resolution
	g, gctx := errgroup.WithContext(ctx)

	for i := 0; i < structType.NumFields(); i++ {
		i := i // Capture for goroutine
		g.Go(func() error {
			field := structType.Field(i)
			fieldType, err := tr.ResolveType(gctx, field.Type())
			if err != nil {
				return fmt.Errorf("failed to resolve field %s: %w", field.Name(), err)
			}

			tag := ""
			if structType.Tag(i) != "" {
				tag = structType.Tag(i)
			}

			fields[i] = &domain.Field{
				Name:      field.Name(),
				Type:      fieldType,
				Tag:       tag,
				Exported:  field.Exported(),
				Embedded:  field.Embedded(),
				Anonymous: field.Anonymous(),
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Convert []*domain.Field to []domain.Field
	fieldSlice := make([]domain.Field, len(fields))
	for i, field := range fields {
		fieldSlice[i] = *field
	}
	return domain.NewStructType("", fieldSlice, ""), nil
}

// resolveInterfaceType handles interface types
func (tr *TypeResolver) resolveInterfaceType(ctx context.Context, iface *types.Interface) (domain.Type, error) {
	methods := make([]*domain.Method, iface.NumMethods())

	for i := 0; i < iface.NumMethods(); i++ {
		method := iface.Method(i)
		_, ok := method.Type().(*types.Signature)
		if !ok {
			return nil, fmt.Errorf("expected signature for method %s", method.Name())
		}

		// Convert signature to domain method
		// This is simplified - full implementation would need more detail
		methods[i] = &domain.Method{
			// Basic method info - full implementation would populate all fields
		}
	}

	return domain.NewInterfaceType(methods), nil
}

// resolveChanType handles channel types
func (tr *TypeResolver) resolveChanType(ctx context.Context, chanType *types.Chan) (domain.Type, error) {
	elem, err := tr.ResolveType(ctx, chanType.Elem())
	if err != nil {
		return nil, err
	}

	direction := tr.mapChanDirection(chanType.Dir())
	return domain.NewChannelType(elem, direction), nil
}

// resolveSignatureType handles function signature types
func (tr *TypeResolver) resolveSignatureType(ctx context.Context, sig *types.Signature) (domain.Type, error) {
	// Resolve parameters
	params := make([]domain.Type, sig.Params().Len())
	for i := 0; i < sig.Params().Len(); i++ {
		param, err := tr.ResolveType(ctx, sig.Params().At(i).Type())
		if err != nil {
			return nil, err
		}
		params[i] = param
	}

	// Resolve returns
	returns := make([]domain.Type, sig.Results().Len())
	for i := 0; i < sig.Results().Len(); i++ {
		result, err := tr.ResolveType(ctx, sig.Results().At(i).Type())
		if err != nil {
			return nil, err
		}
		returns[i] = result
	}

	return domain.NewFunctionType(params, returns, sig.Variadic()), nil
}

// resolveTypeParam handles type parameters (generics)
func (tr *TypeResolver) resolveTypeParam(ctx context.Context, param *types.TypeParam) (domain.Type, error) {
	constraint, err := tr.ResolveType(ctx, param.Constraint())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve type parameter constraint: %w", err)
	}

	return domain.NewTypeParameterType(param.Obj().Name(), constraint), nil
}

// analyzeTypeStructure creates detailed type information for field mapping
func (p *ASTParser) analyzeTypeStructure(ctx context.Context, domainType domain.Type) (*domain.TypeInfo, error) {
	// Check if already cached
	if cached, ok := p.typeResolverPool.resolvers[0].typeInfoMap.Load(domainType.Name()); ok {
		return cached.(*domain.TypeInfo), nil
	}

	var typeInfo *domain.TypeInfo

	switch domainType.Kind() {
	case domain.KindStruct:
		typeInfo = p.analyzeStructTypeInfo(ctx, domainType)
	case domain.KindPointer:
		// Analyze the pointed-to type
		if pointerType, ok := domainType.(*domain.PointerType); ok {
			return p.analyzeTypeStructure(ctx, pointerType.Elem())
		}
	case domain.KindSlice:
		typeInfo = p.analyzeCollectionTypeInfo(ctx, domainType)
	case domain.KindMap:
		typeInfo = p.analyzeMapTypeInfo(ctx, domainType)
	default:
		// For basic types, create simple type info
		typeInfo = &domain.TypeInfo{
			Name:       domainType.Name(),
			Kind:       domainType.Kind(),
			Fields:     nil,
			Methods:    nil,
			TypeParams: domainType.TypeParams(),
		}
	}

	// Cache the result
	p.typeResolverPool.resolvers[0].typeInfoMap.Store(domainType.Name(), typeInfo)

	return typeInfo, nil
}

// Helper methods

func (tr *TypeResolver) mapBasicTypeKind(kind types.BasicKind) reflect.Kind {
	switch kind {
	case types.Bool:
		return reflect.Bool
	case types.Int:
		return reflect.Int
	case types.Int8:
		return reflect.Int8
	case types.Int16:
		return reflect.Int16
	case types.Int32:
		return reflect.Int32
	case types.Int64:
		return reflect.Int64
	case types.Uint:
		return reflect.Uint
	case types.Uint8:
		return reflect.Uint8
	case types.Uint16:
		return reflect.Uint16
	case types.Uint32:
		return reflect.Uint32
	case types.Uint64:
		return reflect.Uint64
	case types.Float32:
		return reflect.Float32
	case types.Float64:
		return reflect.Float64
	case types.String:
		return reflect.String
	default:
		return reflect.Interface
	}
}

func (tr *TypeResolver) mapChanDirection(dir types.ChanDir) domain.ChannelDirection {
	switch dir {
	case types.SendRecv:
		return domain.ChannelBidirectional
	case types.SendOnly:
		return domain.ChannelSendOnly
	case types.RecvOnly:
		return domain.ChannelReceiveOnly
	default:
		return domain.ChannelBidirectional
	}
}

func (p *ASTParser) analyzeStructTypeInfo(ctx context.Context, domainType domain.Type) *domain.TypeInfo {
	structType, ok := domainType.(*domain.StructType)
	if !ok {
		return nil
	}

	// Get fields and convert to pointer slice
	fields := structType.Fields()
	fieldPtrs := make([]*domain.Field, len(fields))
	for i := range fields {
		fieldPtrs[i] = &fields[i]
	}

	return &domain.TypeInfo{
		Name:       domainType.Name(),
		Kind:       domainType.Kind(),
		Fields:     fieldPtrs,
		Methods:    nil, // Would be populated if needed
		TypeParams: domainType.TypeParams(),
	}
}

func (p *ASTParser) analyzeCollectionTypeInfo(ctx context.Context, domainType domain.Type) *domain.TypeInfo {
	return &domain.TypeInfo{
		Name:       domainType.Name(),
		Kind:       domainType.Kind(),
		Fields:     nil,
		Methods:    nil,
		TypeParams: domainType.TypeParams(),
	}
}

func (p *ASTParser) analyzeMapTypeInfo(ctx context.Context, domainType domain.Type) *domain.TypeInfo {
	return &domain.TypeInfo{
		Name:       domainType.Name(),
		Kind:       domainType.Kind(),
		Fields:     nil,
		Methods:    nil,
		TypeParams: domainType.TypeParams(),
	}
}
