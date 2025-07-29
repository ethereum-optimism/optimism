package clients

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFourByteClient_LookupBy4ByteDirectory(t *testing.T) {

	mockResponse := FourByteResponse{
		Count:    1,
		Next:     nil,
		Previous: nil,
		Results: []FourByteResult{
			{
				ID:             1,
				CreatedAt:      "2023-01-01T00:00:00Z",
				TextSignature:  "transfer(address,uint256)",
				HexSignature:   "0xa9059cbb",
				BytesSignature: "a9059cbb",
			},
		},
	}

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径和参数
		expectedPath := "/api/v1/signatures/"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		if r.URL.Query().Get("hex_signature") != "0xa9059cbb" {
			t.Errorf("Expected hex_signature=0xa9059cbb, got %s", r.URL.Query().Get("hex_signature"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewFourByteClient()
	client.url = server.URL + "/api/v1/signatures/"

	signatures, err := client.LookupBy4ByteDirectory("0xa9059cbb")

	if err != nil {
		t.Fatalf("LookupBy4ByteDirectory() error = %v", err)
	}

	if len(signatures) == 0 {
		t.Fatal("Expected at least one signature, got none")
	}

	found := false
	for _, sig := range signatures {
		if sig == "transfer(address,uint256)" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected to find 'transfer(address,uint256)' in results, got %v", signatures)
	}
}
