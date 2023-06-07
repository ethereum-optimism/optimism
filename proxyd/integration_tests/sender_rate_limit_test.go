package integration_tests

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/proxyd"
	"github.com/stretchr/testify/require"
)

const txHex1 = "0x02f8b28201a406849502f931849502f931830147f9948f3ddd0fbf3e78ca1d6c" +
	"d17379ed88e261249b5280b84447e7ef2400000000000000000000000089c8b1" +
	"b2774201bac50f627403eac1b732459cf7000000000000000000000000000000" +
	"0000000000000000056bc75e2d63100000c080a0473c95566026c312c9664cd6" +
	"1145d2f3e759d49209fe96011ac012884ec5b017a0763b58f6fa6096e6ba28ee" +
	"08bfac58f58fb3b8bcef5af98578bdeaddf40bde42"

const txHex2 = "0xf8aa82afd2830f4240830493e094464959ad46e64046b891f562cff202a465d5" +
	"22f380b844d5bade070000000000000000000000004200000000000000000000" +
	"0000000000000000060000000000000000000000000000000000000000000000" +
	"0000000025ef43fc0038a05d8ea9837ea81469bda4dadbe852fdd37fcfbcd666" +
	"5641a35e4726fbc04364e7a0107e20bb34aea53c695a551204a11d42fe465055" +
	"510ee240e8f884fb70289be6"

const dummyRes = `{"id": 123, "jsonrpc": "2.0", "result": "dummy"}`

const limRes = `{"error":{"code":-32017,"message":"sender is over rate limit"},"id":1,"jsonrpc":"2.0"}`

func TestSenderRateLimitValidation(t *testing.T) {
	goodBackend := NewMockBackend(SingleResponseHandler(200, dummyRes))
	defer goodBackend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", goodBackend.URL()))

	config := ReadConfig("sender_rate_limit")

	// Don't perform rate limiting in this test since we're only testing
	// validation.
	config.SenderRateLimit.Limit = math.MaxInt
	client := NewProxydClient("http://127.0.0.1:8545")
	_, shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	f, err := os.Open("testdata/testdata.txt")
	require.NoError(t, err)
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan() // skip header
	for scanner.Scan() {
		record := strings.Split(scanner.Text(), "|")
		name, body, expResponseBody := record[0], record[1], record[2]
		require.NoError(t, err)
		t.Run(name, func(t *testing.T) {
			res, _, err := client.SendRequest([]byte(body))
			require.NoError(t, err)
			RequireEqualJSON(t, []byte(expResponseBody), res)
		})
	}
}

func TestSenderRateLimitLimiting(t *testing.T) {
	goodBackend := NewMockBackend(SingleResponseHandler(200, dummyRes))
	defer goodBackend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", goodBackend.URL()))

	config := ReadConfig("sender_rate_limit")
	client := NewProxydClient("http://127.0.0.1:8545")
	_, shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	// Two separate requests from the same sender
	// should be rate limited.
	res1, code1, err := client.SendRequest(makeSendRawTransaction(txHex1))
	require.NoError(t, err)
	res2, code2, err := client.SendRequest(makeSendRawTransaction(txHex1))
	require.NoError(t, err)
	RequireEqualJSON(t, []byte(dummyRes), res1)
	require.Equal(t, 200, code1)
	RequireEqualJSON(t, []byte(limRes), res2)
	require.Equal(t, 429, code2)

	// Clear the limiter.
	time.Sleep(1100 * time.Millisecond)

	// Two separate requests from different senders
	// should not be rate limited.
	res1, code1, err = client.SendRequest(makeSendRawTransaction(txHex1))
	require.NoError(t, err)
	res2, code2, err = client.SendRequest(makeSendRawTransaction(txHex2))
	require.NoError(t, err)
	RequireEqualJSON(t, []byte(dummyRes), res1)
	require.Equal(t, 200, code1)
	RequireEqualJSON(t, []byte(dummyRes), res2)
	require.Equal(t, 200, code2)

	// Clear the limiter.
	time.Sleep(1100 * time.Millisecond)

	// A batch request should rate limit within the batch itself.
	batch := []byte(fmt.Sprintf(
		`[%s, %s, %s]`,
		makeSendRawTransaction(txHex1),
		makeSendRawTransaction(txHex1),
		makeSendRawTransaction(txHex2),
	))
	res, code, err := client.SendRequest(batch)
	require.NoError(t, err)
	require.Equal(t, 200, code)
	RequireEqualJSON(t, []byte(fmt.Sprintf(
		`[%s, %s, %s]`,
		dummyRes,
		limRes,
		dummyRes,
	)), res)
}

func makeSendRawTransaction(dataHex string) []byte {
	return []byte(`{"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":["` + dataHex + `"],"id":1}`)
}
