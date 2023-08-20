package serializers

import (
	"context"
	"errors"
	"fmt"
	"reflect"

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
	// Empty slices are serialized as '0x'
	if dbValue == nil {
		return errors.New("cannot unmarshal an empty dbValue")
	}

	hexStr, ok := dbValue.(string)
	if !ok {
		return fmt.Errorf("expected hex string as the database value: %T", dbValue)
	}

	b, err := hexutil.Decode(hexStr)
	if err != nil {
		return fmt.Errorf("failed to decode database value: %s", err)
	}

	fieldValue := reflect.New(field.FieldType)
	if field.FieldType.Kind() == reflect.Pointer {
		// Allocate memory if this is pointer which by
		// default when deserializing is probably `nil`
		fieldValue.Set(reflect.New(field.FieldType.Elem()))
	}

	fieldInterface := fieldValue.Interface()
	fieldSetBytes, ok := fieldInterface.(SetBytesInterface)
	if !ok {
		return fmt.Errorf("field does not satisfy the `SetBytes([]byte)` interface: %T", fieldInterface)
	}

	fieldSetBytes.SetBytes(b)
	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())
	return nil
}

func (BytesSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	fieldBytes, ok := fieldValue.(BytesInterface)
	if !ok {
		return nil, fmt.Errorf("field does not satisfy the `Bytes() []byte` interface")
	}

	var b []byte
	if fieldValue != nil && reflect.ValueOf(fieldValue).IsNil() {
		b = fieldBytes.Bytes()
	}

	hexStr := hexutil.Encode(b)
	return hexStr, nil
}
