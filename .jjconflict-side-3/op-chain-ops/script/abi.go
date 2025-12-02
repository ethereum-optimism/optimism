package script

import (
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// matchTypes is a runtime ABI type check utility that ensures that compile-time structs
// match the ABI definition loaded from an artifact at runtime
//
// This verification is important since even with abigen-generated types the ABI can deviate
// which would cause a lot of headache and e.g. partially successful deployments or configurations
func matchTypes(abiType abi.Type, goType reflect.Type) error {
	// If the types are convertible, we're good
	if goType.AssignableTo(abiType.GetType()) {
		return nil
	}

	// We check for arrays first (i.e. fixed length slices like uint256[2])
	if abiType.T == abi.ArrayTy {
		// First a basic check
		if goType.Kind() != reflect.Array {
			return abiTypeErr(abiType, goType)
		}

		// Now make sure the lengths match
		if abiType.Size != goType.Len() {
			return fmt.Errorf("%w: expected an array of length %d, got length %d", abiTypeErr(abiType, goType), abiType.Size, goType.Len())
		}

		// Finally we check the element types
		err := matchTypes(*abiType.Elem, goType.Elem())
		if err != nil {
			return fmt.Errorf("%w: %w", abiTypeErr(abiType, goType), err)
		}

		// If all the checks above succeeded, it means the array is safe to be used
		return nil
	}

	// Now we check for slice type (i.e. variable length slices like uint256[])
	if abiType.T == abi.SliceTy {
		// First a basic check
		if goType.Kind() != reflect.Slice {
			return abiTypeErr(abiType, goType)
		}

		// Then check the element types
		err := matchTypes(*abiType.Elem, goType.Elem())
		if err != nil {
			return fmt.Errorf("%w: %w", abiTypeErr(abiType, goType), err)
		}

		// If all the checks above succeeded, it means the slice is safe to be used
		return nil
	}

	// Finally the most complex ones, tuples
	if abiType.T == abi.TupleTy {
		// First a basic check
		if goType.Kind() != reflect.Struct {
			return abiTypeErr(abiType, goType)
		}

		// Then we compare the number of fields
		numAbiFields := abiType.TupleType.NumField()
		numGoFields := goType.NumField()
		if numAbiFields != numGoFields {
			return fmt.Errorf("%w: the number of struct fields doesn't match: ABI type has %d, Go type has %d", abiTypeErr(abiType, goType), numAbiFields, numGoFields)
		}

		// And finally we check each field
		for index := range numAbiFields {
			field := abiType.TupleType.Field(index)
			goField := goType.Field(index)

			// First we make sure that the names are sorted in the correct order
			//
			// This is important since ABI encoding and decoding specifically has issues
			// with misordered fields and can place values in wrong places
			//
			// Here we need to take `abi:` tags into consideration since if present,
			// they will dictate the ABI <-> Go field mapping instead of the struct names
			goFieldTagName, ok := goField.Tag.Lookup("abi")
			if ok {
				// If the tag is present, we'll match it with the corresponding ABI tuple type name
				abiFieldRawName := abiType.TupleRawNames[index]
				if goFieldTagName != abiFieldRawName {
					return fmt.Errorf("%w: ABI field name %s at index %d does not match Go field name %s. Please make sure to match the Go structs with Solidity structs", abiTypeErr(abiType, goType), field.Name, index, goField.Name)
				}
			} else {
				// If there is no `abi:` tag, we'll match the field names themselves
				if field.Name != goField.Name {
					return fmt.Errorf("%w: ABI field name %s at index %d does not match Go field name %s. Please make sure to match the Go structs with Solidity structs", abiTypeErr(abiType, goType), field.Name, index, goField.Name)
				}
			}

			// Now we ensure that the types match
			err := matchTypes(*abiType.TupleElems[index], goField.Type)
			if err != nil {
				return fmt.Errorf("%w: ABI field %s does not match Go field %s: %w", abiTypeErr(abiType, goType), field.Name, goField.Name, err)
			}
		}

		// If all the checks above succeeded, it means the tuple is safe to be used
		return nil
	}

	// We'll return a default error
	return abiTypeErr(abiType, goType)
}

// matchArguments ensures that an argument list (e.g. function argument or return values)
// match the provided Go types
func matchArguments(args abi.Arguments, goTypes ...reflect.Type) error {
	// First we make sure that the argument lengths match
	numAbiArgs := len(args)
	numGoTypes := len(goTypes)
	if numAbiArgs != numGoTypes {
		return fmt.Errorf("ABI arguments don't match Go types: ABI has %d arguments, Go has %d", numAbiArgs, numGoTypes)
	}

	for index, abiArg := range args {
		goType := goTypes[index]

		err := matchTypes(abiArg.Type, goType)
		if err != nil {
			return fmt.Errorf("ABI argument %s at index %d doesn't match Go type: %w", abiArg.Name, index, err)
		}
	}

	return nil
}

func abiTypeErr(abiType abi.Type, goType reflect.Type) error {
	return fmt.Errorf("ABI type %s (represented by %s) is not assignable to Go type %s", abiType, abiType.GetType(), goType)
}
