package database

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"

	"gorm.io/gorm/schema"
)

type RLPSerializer struct{}

type RLPInterface interface {
	rlp.Encoder
	rlp.Decoder
}

func init() {
	schema.RegisterSerializer("rlp", RLPSerializer{})
}

func (RLPSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) error {
	fieldValue := reflect.New(field.FieldType)
	if dbValue != nil {
		var bytes []byte
		switch v := dbValue.(type) {
		case []byte:
			bytes = v
		case string:
			b, err := hexutil.Decode(v)
			if err != nil {
				return err
			}
			bytes = b
		default:
			return fmt.Errorf("unrecognized RLP bytes: %#v", dbValue)
		}

		if len(bytes) > 0 {
			err := rlp.DecodeBytes(bytes, fieldValue.Interface())
			if err != nil {
				return err
			}
		}
	}

	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())
	return nil
}

func (RLPSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	// Even though rlp.Encode takes an interface and will error out if the passed interface does not
	// satisfy the interface, we check here since we also want to make sure this type satisfies the
	// rlp.Decoder interface as well
	i := reflect.TypeOf(new(RLPInterface)).Elem()
	if !reflect.TypeOf(fieldValue).Implements(i) {
		return nil, fmt.Errorf("%T does not satisfy RLP Encoder & Decoder interface", fieldValue)
	}

	rlpBytes, err := rlp.EncodeToBytes(fieldValue)
	if err != nil {
		return nil, err
	}

	return hexutil.Bytes(rlpBytes).MarshalText()
}
