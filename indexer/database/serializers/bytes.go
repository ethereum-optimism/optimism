package serializers

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"gorm.io/gorm/schema"
)

type BytesSerializer struct{}
type BytesInterface interface{ Bytes() []byte }
type SetBytesInterface interface{ SetBytes([]byte) }

func init() {
	schema.RegisterSerializer("bytes", BytesSerializer{})
}

func (BytesSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) error {
	if dbValue == nil {
		return nil
	}

	hexStr, ok := dbValue.(string)
	if !ok {
		return fmt.Errorf("expected hex string as the database value: %T", dbValue)
	}

	b, err := hexutil.Decode(hexStr)
	if err != nil {
		return fmt.Errorf("failed to decode database value: %w", err)
	}

	fieldValue := reflect.New(field.FieldType)
	fieldInterface := fieldValue.Interface()

	// Detect if we're deserializing into a pointer. If so, we'll need to
	// also allocate memory to where the allocated pointer should point to
	if field.FieldType.Kind() == reflect.Pointer {
		nestedField := fieldValue.Elem()
		if nestedField.Elem().Kind() == reflect.Pointer {
			return fmt.Errorf("double pointers are the max depth supported: %T", fieldValue)
		}

		// We'll want to call `SetBytes` on the pointer to
		// the allocated memory and not the double pointer
		nestedField.Set(reflect.New(field.FieldType.Elem()))
		fieldInterface = nestedField.Interface()
	}

	fieldSetBytes, ok := fieldInterface.(SetBytesInterface)
	if !ok {
		return fmt.Errorf("field does not satisfy the `SetBytes([]byte)` interface: %T", fieldInterface)
	}

	fieldSetBytes.SetBytes(b)
	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())
	return nil
}

func (BytesSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	if fieldValue == nil || (field.FieldType.Kind() == reflect.Pointer && reflect.ValueOf(fieldValue).IsNil()) {
		return nil, nil
	}

	fieldBytes, ok := fieldValue.(BytesInterface)
	if !ok {
		return nil, fmt.Errorf("field does not satisfy the `Bytes() []byte` interface")
	}

	hexStr := hexutil.Encode(fieldBytes.Bytes())
	return strings.ToLower(hexStr), nil
}
