package serialize

import (
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
)

func Load[X any](inputPath string) (*X, error) {
	if IsBinaryState(inputPath) {
		return LoadSerializedBinary[X](inputPath)
	}
	return jsonutil.LoadJSON[X](inputPath)
}

func Write[X Serializable](outputPath string, x X, perm os.FileMode) error {
	if IsBinaryState(outputPath) {
		return WriteSerializedBinary(x, ioutil.ToStdOutOrFileOrNoop(outputPath, perm))
	}
	return jsonutil.WriteJSON[X](x, ioutil.ToStdOutOrFileOrNoop(outputPath, perm))
}

func IsBinaryState(path string) bool {
	return strings.HasSuffix(path, ".bin") || strings.HasSuffix(path, ".bin.gz")
}
