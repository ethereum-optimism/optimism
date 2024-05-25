package serializers

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"

	"gorm.io/gorm/schema"
)

type RLPSerializer struct{}

func init() {
	schema.RegisterSerializer("rlp", RLPSerializer{})
}

func (RLPSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) error {
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
	if err := rlp.DecodeBytes(b, fieldValue.Interface()); err != nil {
		return fmt.Errorf("failed to decode rlp bytes: %w", err)
	}

	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())
	return nil
}

func (RLPSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	if fieldValue == nil || (field.FieldType.Kind() == reflect.Pointer && reflect.ValueOf(fieldValue).IsNil()) {
		return nil, nil
	}

	rlpBytes, err := rlp.EncodeToBytes(fieldValue)
	if err != nil {
		return nil, fmt.Errorf("failed to encode rlp bytes: %w", err)
	}

	hexStr := hexutil.Encode(rlpBytes)
	return strings.ToLower(hexStr), nil
}
