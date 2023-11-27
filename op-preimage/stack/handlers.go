package stack

import (
	"errors"
	"io"
	"io/fs"

	"github.com/ethereum/go-ethereum/log"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	oppio "github.com/ethereum-optimism/optimism/op-program/io"
)

func HintForwarder(hintCh oppio.FileChannel) preimage.HintHandler {
	hintCl := preimage.NewHintWriter(hintCh)
	return func(hint string) error {
		hintCl.Hint(preimage.RawHint(hint))
		return nil
	}
}

func HandleHints(logger log.Logger, hintCh io.ReadWriter, hinter preimage.HintHandler) error {
	hintReader := preimage.NewHintReader(hintCh)
	for {
		if err := hintReader.NextHint(hinter); err != nil {
			if err == io.EOF || errors.Is(err, fs.ErrClosed) {
				logger.Debug("closing pre-image hint handler")
				return nil
			}
			logger.Error("pre-image hint router error", "err", err)
			return err
		}
	}
}

func PreimageForwarder(preimageCh oppio.FileChannel) preimage.PreimageGetter {
	preimageCl := preimage.NewOracleClient(preimageCh)
	return func(key [32]byte) ([]byte, error) {
		return preimageCl.Get(preimage.RawKey(key)), nil
	}
}

func HandlePreimages(logger log.Logger, preimageCh io.ReadWriteCloser, getter preimage.PreimageGetter) error {
	server := preimage.NewOracleServer(preimageCh)
	for {
		if err := server.NextPreimageRequest(getter); err != nil {
			if err == io.EOF || errors.Is(err, fs.ErrClosed) {
				logger.Debug("closing pre-image server")
				return nil
			}
			logger.Error("pre-image server error", "error", err)
			return err
		}
	}
}
