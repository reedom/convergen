//go:build convergen

package multi_intf

type DomainModel struct {
	ID string
}

type TransportModel struct {
	ID string
}

type StorageModel struct {
	ID string
}

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :recv d
	ToTransport(*DomainModel) *TransportModel
	// :recv d
	ToStorage(*DomainModel) *StorageModel
}

// :convergen
type StorageConverter interface {
	// :recv s
	ToTransport(*StorageModel) *TransportModel
	// :recv s
	ToDomain(*StorageModel) *DomainModel
}
