//go:build convergen

package slice

type SrcType struct {
	IntSlice    []int
	DataSlice   []Data
	StatusSlice []int
}

type DstType struct {
	IntSlice    []int
	DataSlice   []Data
	StatusSlice []Status
}

type Data struct{}

type Status int

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :typecast
	Copy(*SrcType) *DstType
}
