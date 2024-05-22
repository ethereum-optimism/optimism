package subkey

import (
	"encoding/binary"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/blake2b"
)

const (
	// DevPhrase is default phrase used for dev test accounts
	DevPhrase = "bottom drive obey lake curtain smoke basket hold race lonely fit walk"

	junctionIDLen = 32
)

var (
	re = regexp.MustCompile(`^(?P<phrase>[\d\w ]+)?(?P<path>(//?[^/]+)*)(///(?P<password>.*))?$`)

	reJunction = regexp.MustCompile(`/(/?[^/]+)`)
)

type DeriveJunction struct {
	ChainCode [32]byte
	IsHard    bool
}

func deriveJunctions(codes []string) (djs []DeriveJunction, err error) {
	for _, code := range codes {
		dj, err := parseDeriveJunction(code)
		if err != nil {
			return nil, err
		}

		djs = append(djs, dj)
	}

	return djs, nil
}

func parseDeriveJunction(code string) (DeriveJunction, error) {
	var jd DeriveJunction
	if strings.HasPrefix(code, "/") {
		jd.IsHard = true
		code = strings.TrimPrefix(code, "/")
	}

	var bc []byte
	u64, err := strconv.ParseUint(code, 10, 0)
	if err == nil {
		bc = make([]byte, 8)
		binary.LittleEndian.PutUint64(bc, u64)
	} else {
		cl, err := compactUint(uint64(len(code)))
		if err != nil {
			return jd, err
		}

		bc = append(cl, code...)
	}

	if len(bc) > junctionIDLen {
		b := blake2b.Sum256(bc)
		bc = b[:]
	}

	copy(jd.ChainCode[:len(bc)], bc)
	return jd, nil
}

func derivePath(path string) (parts []string) {
	res := reJunction.FindAllStringSubmatch(path, -1)
	for _, p := range res {
		parts = append(parts, p[1])
	}
	return parts
}

func splitURI(suri string) (phrase string, pathMap string, password string, err error) {
	res := re.FindStringSubmatch(suri)
	if res == nil {
		return phrase, pathMap, password, errors.New("invalid URI format")
	}

	phrase = res[1]
	if phrase == "" {
		phrase = DevPhrase
	}

	return phrase, res[2], res[5], nil
}
