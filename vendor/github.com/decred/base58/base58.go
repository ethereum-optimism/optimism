// Copyright (c) 2013-2015 The btcsuite developers
// Copyright (c) 2015-2020 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package base58

//go:generate go run genalphabet.go

// Decode decodes a modified base58 string to a byte slice.
func Decode(input string) []byte {
	if len(input) == 0 {
		return []byte("")
	}

	// The max possible output size is when a base58 encoding consists of
	// nothing but the alphabet character at index 0 which would result in the
	// same number of bytes as the number of input chars.
	output := make([]byte, len(input))

	// Encode to base256 in reverse order to avoid extra calculations to
	// determine the final output size in favor of just keeping track while
	// iterating.
	var index int
	for _, r := range []byte(input) {
		// Invalid base58 character.
		val := uint32(b58[r])
		if val == 255 {
			return []byte("")
		}

		// Multiply each byte in the output by 58 and encode to base256 while
		// propagating the carry.
		for i, b := range output[:index] {
			val += uint32(b) * 58
			output[i] = byte(val)
			val >>= 8
		}
		for ; val > 0; val >>= 8 {
			output[index] = byte(val)
			index++
		}
	}

	// Account for the leading zeros in the input.  They are appended since the
	// encoding is happening in reverse order.
	for _, r := range []byte(input) {
		if r != alphabetIdx0 {
			break
		}

		output[index] = 0
		index++
	}

	// Truncate the output buffer to the actual number of decoded bytes and
	// reverse it since it was calculated in reverse order.
	output = output[:index:index]
	for i := 0; i < index/2; i++ {
		output[i], output[index-1-i] = output[index-1-i], output[i]
	}

	return output
}

// Encode encodes a byte slice to a modified base58 string.
func Encode(input []byte) string {
	// Since the conversion is from base256 to base58, the max possible number
	// of bytes of output per input byte is log_58(256) ~= 1.37.  Thus, the max
	// total output size is ceil(len(input) * 137/100).  Rather than worrying
	// about the ceiling, just add one even if it isn't needed since the final
	// output is truncated to the right size at the end.
	output := make([]byte, (len(input)*137/100)+1)

	// Encode to base58 in reverse order to avoid extra calculations to
	// determine the final output size in favor of just keeping track while
	// iterating.
	var index int
	for _, r := range input {
		// Multiply each byte in the output by 256 and encode to base58 while
		// propagating the carry.
		val := uint32(r)
		for i, b := range output[:index] {
			val += uint32(b) << 8
			output[i] = byte(val % 58)
			val /= 58
		}
		for ; val > 0; val /= 58 {
			output[index] = byte(val % 58)
			index++
		}
	}

	// Replace the calculated remainders with their corresponding base58 digit.
	for i, b := range output[:index] {
		output[i] = alphabet[b]
	}

	// Account for the leading zeros in the input.  They are appended since the
	// encoding is happening in reverse order.
	for _, r := range input {
		if r != 0 {
			break
		}

		output[index] = alphabetIdx0
		index++
	}

	// Truncate the output buffer to the actual number of encoded bytes and
	// reverse it since it was calculated in reverse order.
	output = output[:index:index]
	for i := 0; i < index/2; i++ {
		output[i], output[index-1-i] = output[index-1-i], output[i]
	}

	return string(output)
}
