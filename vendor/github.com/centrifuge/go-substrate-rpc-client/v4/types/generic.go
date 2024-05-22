package types

import "github.com/centrifuge/go-substrate-rpc-client/v4/scale"

type Option[T any] struct {
	hasValue bool
	value    T
}

func NewOption[T any](t T) Option[T] {
	return Option[T]{hasValue: true, value: t}
}

func NewEmptyOption[T any]() Option[T] {
	return Option[T]{hasValue: false}
}

func (o *Option[T]) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

func (o Option[T]) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

// SetSome sets a value
func (o *Option[T]) SetSome(value T) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *Option[T]) SetNone() {
	o.hasValue = false

	var val T

	o.value = val
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o *Option[T]) Unwrap() (ok bool, value T) {
	return o.hasValue, o.value
}

func (o *Option[T]) HasValue() bool {
	return o.hasValue
}
