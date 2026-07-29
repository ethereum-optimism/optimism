package addresses

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
)

var ErrZeroAddress = errors.New("found zero address")
var ErrNilPointer = errors.New("nil pointer provided")
var ErrNotAddressType = errors.New("field is not of type common.Address")

// CheckNoZeroAddresses checks that all fields in a struct are of type common.Address
// and that none of them are zero addresses. Works with both struct values and pointers to structs.
// Returns an error if any field is not a common.Address or if any common.Address field is zero.
func CheckNoZeroAddresses(s interface{}) error {
	val := reflect.ValueOf(s)

	// If we have a pointer, dereference it to get the struct
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return ErrNilPointer
		}
		val = val.Elem()
	}

	// Ensure we're working with a struct
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("can only check structs, but got %s", val.Kind())
	}

	typ := val.Type()
	addressType := reflect.TypeOf(common.Address{})

	// Iterate through all the fields
	for i := 0; i < val.NumField(); i++ {
		fieldValue := val.Field(i)
		fieldName := typ.Field(i).Name

		// Verify field is a common.Address
		if fieldValue.Type() != addressType {
			return fmt.Errorf("%w: %s (type: %s)", ErrNotAddressType, fieldName, fieldValue.Type())
		}

		// Check if the address is zero
		if fieldValue.Interface() == (common.Address{}) {
			return fmt.Errorf("%w: %s", ErrZeroAddress, fieldName)
		}
	}
	return nil
}
