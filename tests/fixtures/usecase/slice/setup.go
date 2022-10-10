//go:build convergen

package slice

type SrcType struct {
	IntSlice  []int
	DataSlice []Data
}

type DstType struct {
	IntSlice  []int
	DataSlice []Data
}

type Data struct{}

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	Copy(*SrcType) *DstType
}
