package rollup

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/jarcoal/httpmock"
)

const url = "http://localhost:9999"

func TestRollupClientCannotConnect(t *testing.T) {
	endpoint := fmt.Sprintf("%s/eth/context/latest", url)
	client := NewClient(url, big.NewInt(1))

	httpmock.ActivateNonDefault(client.client.GetClient())

	response, _ := httpmock.NewJsonResponder(
		400,
		map[string]interface{}{},
	)
	httpmock.RegisterResponder(
		"GET",
		endpoint,
		response,
	)

	context, err := client.GetLatestEthContext()
	if context != nil {
		t.Fatal("returned value is not nil")
	}
	if !errors.Is(err, errHTTPError) {
		t.Fatalf("Incorrect error returned: %s", err)
	}
}
func TestDecodedJSON(t *testing.T) {
	str := []byte(`
	{
		"index": 643116,
		"batchIndex": 21083,
		"blockNumber": 25954867,
		"timestamp": 1625605288,
		"gasLimit": "11000000",
		"target": "0x4200000000000000000000000000000000000005",
		"origin": null,
		"data": "0xf86d0283e4e1c08343eab8941a5245ea5210c3b57b7cfdf965990e63534a7b528901a055690d9db800008081aea019f7c6719f1718475f39fb9e5a6a897c3bd5057488a014666e5ad573ec71cf0fa008836030e686f3175dd7beb8350809b47791c23a19092a8c2fab1f0b4211a466",
		"queueOrigin": "sequencer",
		"value": "0x1a055690d9db80000",
		"queueIndex": null,
		"decoded": {
			"nonce": "2",
			"gasPrice": "15000000",
			"gasLimit": "4451000",
			"value": "0x1a055690d9db80000",
			"target": "0x1a5245ea5210c3b57b7cfdf965990e63534a7b52",
			"data": "0x",
			"sig": {
				"v": 1,
				"r": "0x19f7c6719f1718475f39fb9e5a6a897c3bd5057488a014666e5ad573ec71cf0f",
				"s": "0x08836030e686f3175dd7beb8350809b47791c23a19092a8c2fab1f0b4211a466"
			}
		},
		"confirmed": true
	}`)

	tx := new(transaction)
	json.Unmarshal(str, tx)
	cmp, _ := new(big.Int).SetString("1a055690d9db80000", 16)
	if tx.Value.ToInt().Cmp(cmp) != 0 {
		t.Fatal("Cannot decode")
	}
}
