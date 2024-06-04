package flags

import "fmt"

type DataAvailabilityType string

const (
	// data availability types
	CalldataType DataAvailabilityType = "calldata"
	BlobsType    DataAvailabilityType = "blobs"
)

var DataAvailabilityTypes = []DataAvailabilityType{
	CalldataType,
	BlobsType,
}

func (kind DataAvailabilityType) String() string {
	return string(kind)
}

func (kind *DataAvailabilityType) Set(value string) error {
	if !ValidDataAvailabilityType(DataAvailabilityType(value)) {
		return fmt.Errorf("unknown data-availability type: %q", value)
	}
	*kind = DataAvailabilityType(value)
	return nil
}

func (kind *DataAvailabilityType) Clone() any {
	cpy := *kind
	return &cpy
}

func ValidDataAvailabilityType(value DataAvailabilityType) bool {
	for _, k := range DataAvailabilityTypes {
		if k == value {
			return true
		}
	}
	return false
}
