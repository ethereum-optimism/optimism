package multistream

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
)

// ErrNotSupported is the error returned when the muxer doesn't support
// the protocols tried for the handshake.
type ErrNotSupported[T StringLike] struct {

	// Slice of protocols that were not supported by the muxer
	Protos []T
}

func (e ErrNotSupported[T]) Error() string {
	return fmt.Sprintf("protocols not supported: %v", e.Protos)
}

func (e ErrNotSupported[T]) Is(target error) bool {
	_, ok := target.(ErrNotSupported[T])
	return ok
}

// ErrNoProtocols is the error returned when the no protocols have been
// specified.
var ErrNoProtocols = errors.New("no protocols specified")

// SelectProtoOrFail performs the initial multistream handshake
// to inform the muxer of the protocol that will be used to communicate
// on this ReadWriteCloser. It returns an error if, for example,
// the muxer does not know how to handle this protocol.
func SelectProtoOrFail[T StringLike](proto T, rwc io.ReadWriteCloser) (err error) {
	defer func() {
		if rerr := recover(); rerr != nil {
			fmt.Fprintf(os.Stderr, "caught panic: %s\n%s\n", rerr, debug.Stack())
			err = fmt.Errorf("panic selecting protocol: %s", rerr)
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		var buf bytes.Buffer
		if err := delitmWriteAll(&buf, []byte(ProtocolID), []byte(proto)); err != nil {
			errCh <- err
			return
		}
		_, err := io.Copy(rwc, &buf)
		errCh <- err
	}()
	// We have to read *both* errors.
	err1 := readMultistreamHeader(rwc)
	err2 := readProto(proto, rwc)
	if werr := <-errCh; werr != nil {
		return werr
	}
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}

// SelectOneOf will perform handshakes with the protocols on the given slice
// until it finds one which is supported by the muxer.
func SelectOneOf[T StringLike](protos []T, rwc io.ReadWriteCloser) (proto T, err error) {
	defer func() {
		if rerr := recover(); rerr != nil {
			fmt.Fprintf(os.Stderr, "caught panic: %s\n%s\n", rerr, debug.Stack())
			err = fmt.Errorf("panic selecting one of protocols: %s", rerr)
		}
	}()

	if len(protos) == 0 {
		return "", ErrNoProtocols
	}

	// Use SelectProtoOrFail to pipeline the /multistream/1.0.0 handshake
	// with an attempt to negotiate the first protocol. If that fails, we
	// can continue negotiating the rest of the protocols normally.
	//
	// This saves us a round trip.
	switch err := SelectProtoOrFail(protos[0], rwc); err.(type) {
	case nil:
		return protos[0], nil
	case ErrNotSupported[T]: // try others
	default:
		return "", err
	}
	proto, err = selectProtosOrFail(protos[1:], rwc)
	if _, ok := err.(ErrNotSupported[T]); ok {
		return "", ErrNotSupported[T]{protos}
	}
	return proto, err
}

func selectProtosOrFail[T StringLike](protos []T, rwc io.ReadWriteCloser) (T, error) {
	for _, p := range protos {
		err := trySelect(p, rwc)
		switch err := err.(type) {
		case nil:
			return p, nil
		case ErrNotSupported[T]:
		default:
			return "", err
		}
	}
	return "", ErrNotSupported[T]{protos}
}

func readMultistreamHeader(r io.Reader) error {
	tok, err := ReadNextToken[string](r)
	if err != nil {
		return err
	}

	if tok != ProtocolID {
		return errors.New("received mismatch in protocol id")
	}
	return nil
}

func trySelect[T StringLike](proto T, rwc io.ReadWriteCloser) error {
	err := delimWriteBuffered(rwc, []byte(proto))
	if err != nil {
		return err
	}
	return readProto(proto, rwc)
}

func readProto[T StringLike](proto T, r io.Reader) error {
	tok, err := ReadNextToken[T](r)
	if err != nil {
		return err
	}

	switch tok {
	case proto:
		return nil
	case "na":
		return ErrNotSupported[T]{[]T{proto}}
	default:
		return fmt.Errorf("unrecognized response: %s", tok)
	}
}
