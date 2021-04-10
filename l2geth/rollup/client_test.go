package rollup

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestRollupClientGetL1GasPrice(t *testing.T) {
	url := "http://localhost:9999"
	endpoint := fmt.Sprintf("%s/eth/gasprice", url)
	// url/chain-id does not matter, we'll mock the responses
	client := NewClient(url, big.NewInt(1))
	// activate the mock
	httpmock.ActivateNonDefault(client.client.GetClient())

	// The API responds with a string value
	expectedGasPrice, _ := new(big.Int).SetString("123132132151245817421893", 10)
	body := map[string]interface{}{
		"gasPrice": expectedGasPrice.String(),
	}
	response, _ := httpmock.NewJsonResponder(
		200,
		body,
	)
	httpmock.RegisterResponder(
		"GET",
		endpoint,
		response,
	)

	gasPrice, err := client.GetL1GasPrice()

	if err != nil {
		t.Fatal("could not get mocked gas price", err)
	}

	if gasPrice.Cmp(expectedGasPrice) != 0 {
		t.Fatal("gasPrice is not parsed properly in the client")
	}
}
