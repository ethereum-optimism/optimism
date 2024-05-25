package serializers

import (
	"context"
	"fmt"
	"math/big"
	"reflect"

	"github.com/jackc/pgtype"
	"gorm.io/gorm/schema"
)

var (
	big10              = big.NewInt(10)
	u256BigIntOverflow = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
)

type U256Serializer struct{}

func init() {
	schema.RegisterSerializer("u256", U256Serializer{})
}

func (U256Serializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) error {
	if dbValue == nil {
		return nil
	} else if field.FieldType != reflect.TypeOf((*big.Int)(nil)) {
		return fmt.Errorf("can only deserialize into a *big.Int: %T", field.FieldType)
	}

	numeric := new(pgtype.Numeric)
	err := numeric.Scan(dbValue)
	if err != nil {
		return err
	}

	bigInt := numeric.Int
	if numeric.Exp > 0 {
		factor := new(big.Int).Exp(big10, big.NewInt(int64(numeric.Exp)), nil)
		bigInt.Mul(bigInt, factor)
	}

	if bigInt.Cmp(u256BigIntOverflow) >= 0 {
		return fmt.Errorf("deserialized number larger than u256 can hold: %s", bigInt)
	}

	field.ReflectValueOf(ctx, dst).Set(reflect.ValueOf(bigInt))
	return nil
}

func (U256Serializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	if fieldValue == nil || (field.FieldType.Kind() == reflect.Pointer && reflect.ValueOf(fieldValue).IsNil()) {
		return nil, nil
	} else if field.FieldType != reflect.TypeOf((*big.Int)(nil)) {
		return nil, fmt.Errorf("can only serialize a *big.Int: %T", field.FieldType)
	}

	numeric := pgtype.Numeric{Int: fieldValue.(*big.Int), Status: pgtype.Present}
	return numeric.Value()
}
