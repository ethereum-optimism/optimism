// Copyright 2018 Jsgenesis
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package subkey

import (
	"bytes"
	"encoding/binary"
	"errors"
)

func compactUint(v uint64) ([]byte, error) {
	// This code was copied over and adapted with many thanks from Joystream/parity-codec-go:withreflect@develop
	var buf bytes.Buffer
	if v < 1<<30 {
		switch {
		case v < 1<<6:
			return []byte{byte(v) << 2}, nil
		case v < 1<<14:
			err := binary.Write(&buf, binary.LittleEndian, uint16(v<<2)+1)
			if err != nil {
				return nil, err
			}
		default:
			err := binary.Write(&buf, binary.LittleEndian, uint32(v<<2)+2)
			if err != nil {
				return nil, err
			}
		}
		return buf.Bytes(), nil
	}

	n := byte(0)
	limit := uint64(1 << 32)
	for v >= limit && limit > 256 { // when overflows, limit will be < 256
		n++
		limit <<= 8
	}
	if n > 4 {
		return nil, errors.New("assertion error: n>4 needed to compact-encode uint64")
	}

	err := buf.WriteByte((n << 2) + 3)
	if err != nil {
		return nil, err
	}

	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	_, err = buf.Write(b[:4+n])
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
