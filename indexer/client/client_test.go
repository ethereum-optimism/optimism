package client

import "testing"

func Test_Client(t *testing.T) {

	cfg := &Config{
		PaginationLimit: 100,
		URL:             "https://localhost:8080",
	}

	ic, err := DialIndexerClient(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Get all withdrawals by address

	withdrawals, err := ic.GetAllWithdrawalsByAddress("0xC64c9c88F28072F9DAa60d371acc08cB5FDb9952")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("withdrawals: %+v", withdrawals)
}
