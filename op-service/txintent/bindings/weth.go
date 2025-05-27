package bindings

import (
	"reflect"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common"
)

type WETHCallFactory struct {
	BaseCallFactory
}

func NewWETHCallFactory(opts ...CallFactoryOption) *WETHCallFactory {
	return &WETHCallFactory{BaseCallFactory: *NewBaseCallFactory(opts...)}
}

func (f *WETHCallFactory) WithDefaultAddr() {
	f.ApplyFactoryOptions(WithTo(common.HexToAddress(predeploys.WETH)))
}

type WETH struct {
	WETHCallFactory

	BalanceOf func(addr common.Address) TypedCall[eth.ETH]              `sol:"balanceOf"`
	Transfer  func(dest common.Address, amount eth.ETH) TypedCall[bool] `sol:"transfer"`
}

func NewWETH(f *WETHCallFactory) *WETH {
	weth := WETH{WETHCallFactory: *f}
	CheckFactoryImpl(weth)

	wethRef := reflect.ValueOf(&weth).Elem()
	wethType := reflect.TypeOf(weth)

	for i := range wethRef.NumField() {
		field := wethType.Field(i)
		fieldType := field.Type
		if fieldType.Kind() == reflect.Func {
			methodName := field.Tag.Get(MethodTagName)

			inputTypes := []reflect.Type{}
			for j := range fieldType.NumIn() {
				inputTypes = append(inputTypes, fieldType.In(j))
			}
			outputType := fieldType.Out(0)

			// outer: func(...args) -> <inner: (func() -> (bytes[], error))>
			// inner: func() -> (bytes[], error)
			funcInput := reflect.FuncOf([]reflect.Type{}, []reflect.Type{reflect.TypeOf([]byte{}), reflect.TypeOf((*error)(nil)).Elem()}, false)
			funcInputWrapper := reflect.FuncOf(inputTypes, []reflect.Type{funcInput}, false)

			encoderLambdaLambda := reflect.MakeFunc(funcInputWrapper, func(argsOuter []reflect.Value) []reflect.Value {
				encoderLambda := reflect.MakeFunc(funcInput, func(argsInner []reflect.Value) []reflect.Value {
					callArgs := make([]any, len(argsOuter))
					for i, a := range argsOuter {
						callArgs[i] = a.Interface()
					}
					v0, v1 := encoder(methodName, callArgs...)

					// guard
					var val0 reflect.Value
					if v0 == nil {
						val0 = reflect.Zero(reflect.TypeOf([]byte{}))
					} else {
						val0 = reflect.ValueOf(v0)
					}
					var val1 reflect.Value
					if v1 == nil {
						val1 = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())
					} else {
						val1 = reflect.ValueOf(v1)
					}

					return []reflect.Value{val0, val1}
				})
				inner := encoderLambda.Interface().(func() ([]byte, error))
				return []reflect.Value{reflect.ValueOf(inner)}
			})

			λ := reflect.MakeFunc(fieldType, func(args []reflect.Value) []reflect.Value {
				innerResults := encoderLambdaLambda.Call(args)
				if len(innerResults) != 1 {
					panic("expected one return value")
				}
				innerλ := innerResults[0].Interface().(func() ([]byte, error))
				wrap := reflect.New(outputType).Elem()
				wrap.FieldByName("MethodName").Set(reflect.ValueOf(methodName))
				wrap.FieldByName("EncodeInputLambda").Set(reflect.ValueOf(innerλ))
				wrap.FieldByName("BaseCallFactory").Set(reflect.ValueOf(&f.BaseCallFactory))
				return []reflect.Value{wrap}
			})
			wethRef.FieldByName(field.Name).Set(λ)
		}
	}
	return &weth
}
