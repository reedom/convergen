package types

type From struct {
	String    string
	StringPtr *string
	Int       int
	IntPtr    *int
	Int64     int64
	Float64   float64
	Struct    struct {
		Field string
	}
	StructPtr *struct {
		Another string
	}
	Any          any
	AnyPtr       *any
	EmptyIntf    interface{}
	EmptyIntfPtr *interface{}
	Intf         interface{ Func() }
	StrArray     [2]string
	StrArrayPtr  *[2]string
	StrPtrArray  [2]*string
	StrSlice     []string
	StrSlicePtr  *[]string
	StrPtrSlice  []*string
	AnySlice     []any
	Map          map[string]string
	MapPtr       *map[string]string
	Chan         chan int
	ChanPtr      *chan int
	Func         func(int) string
	FuncPtr      *func(int) string
}

type To struct {
	String    string
	StringPtr *string
	Int       int
	IntPtr    *int
	Int64     int64
	Float64   float64
	Struct    struct {
		Field string
	}
	StructPtr *struct {
		Another string
	}
	Any          any
	AnyPtr       *any
	EmptyIntf    interface{}
	EmptyIntfPtr *interface{}
	Intf         interface{ Func() }
	StrArray     [2]string
	StrArrayPtr  *[2]string
	StrPtrArray  [2]*string
	StrSlice     []string
	StrSlicePtr  *[]string
	StrPtrSlice  []*string
	AnySlice     []any
	Map          map[string]string
	MapPtr       *map[string]string
	Chan         chan int
	ChanPtr      *chan int
	Func         func(int) string
	FuncPtr      *func(int) string
}

// hoge
//
// !go:generate go run github.com/reedom/convergen
type Convergen interface {
	// convergen:map foo bar
	FromTo(*From) *To
}
