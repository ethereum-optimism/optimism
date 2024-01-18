package eth_test

import (
	"encoding"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

// TestAPIGenesisResponse tests that json unmarshaling a json response from a
// eth/v1/beacon/genesis beacon node call into a APIGenesisResponse object
// fills all exising fields with the expected values, thereby confirming that
// APIGenesisResponse is compatible with the current beacon node API.
// This also confirms that the [sources.L1BeaconClient] correctly parses
// responses from a real beacon node.
func TestAPIGenesisResponse(t *testing.T) {
	var resp eth.APIGenesisResponse
	testBeaconAPIResponse(t, &resp, "eth_v1_beacon_genesis_goerli.json")
}

// TestAPIConfigResponse tests that json unmarshaling a json response from a
// eth/v1/config/spec beacon node call into a APIConfigResponse object
// fills all exising fields with the expected values, thereby confirming that
// APIGenesisResponse is compatible with the current beacon node API.
// This also confirms that the [sources.L1BeaconClient] correctly parses
// responses from a real beacon node.
func TestAPIConfigResponse(t *testing.T) {
	var resp eth.APIConfigResponse
	testBeaconAPIResponse(t, &resp, "eth_v1_config_spec_goerli.json")
}

// TestAPIGetBlobSidecarsResponse tests that json unmarshaling a json response from a
// eth/v1/beacon/blob_sidecars/<X> beacon node call into a APIGetBlobSidecarsResponse object
// fills all exising fields with the expected values, thereby confirming that
// APIGenesisResponse is compatible with the current beacon node API.
// This also confirms that the [sources.L1BeaconClient] correctly parses
// responses from a real beacon node.
func TestAPIGetBlobSidecarsResponse(t *testing.T) {
	require := require.New(t)

	path := filepath.Join("testdata", "eth_v1_beacon_blob_sidecars_7422094_goerli.json")
	jsonStr, err := os.ReadFile(path)
	require.NoError(err)

	var resp eth.APIGetBlobSidecarsResponse
	require.NoError(json.Unmarshal(jsonStr, &resp))

	respJsonStr, err := json.Marshal(&resp)
	require.NoError(err)
	// truncate newline of file
	require.Equal(jsonStr[:len(jsonStr)-1], respJsonStr)
}

// testBeaconAPIResponse tests that json-unmarshaling a Beacon node json response
// read from the provided testfile path into the provided response object works
// and that all fields of the response object are populated with the expected values.
//
// It currently assumes that all Beacon responses have the actual resonse object in
// a single "data" field and it doesn't support nested json response objects.
//
// This test in future proof in the sense that if new fields are added to an API response
// struct which wouldn't be populated by the test jsons, it fails.
func testBeaconAPIResponse(t *testing.T, resp any, testfile string) {
	require := require.New(t)

	path := filepath.Join("testdata", testfile)
	jsonStr, err := os.ReadFile(path)
	require.NoError(err)

	require.NoError(json.Unmarshal(jsonStr, &resp))

	jsonMap := make(map[string]any)
	require.NoError(json.Unmarshal(jsonStr, &jsonMap))
	respDataField, err := structFieldByName(resp, "Data")
	require.NoError(err, "response struct Data field error")

	switch jsonData := jsonMap["data"].(type) {
	case (map[string]any):
		if respDataField.Kind() != reflect.Struct {
			t.Fatalf("unexpected data field type (%T) of response, expected struct", respDataField)
		}
		testCompleteJSONResponse(t, respDataField.Interface(), jsonData)
	case []any:
		if respDataField.Kind() != reflect.Slice {
			t.Fatalf("unexpected data field type (%T) of response, expected slice", respDataField)
		}
		require.Len(jsonData, respDataField.Len(), "number of data objects in response and json differ")

		for i, jsonDataElem := range jsonData {
			jsonel, ok := jsonDataElem.(map[string]any)
			require.Truef(ok, "unexpected json data array element type %T", jsonDataElem)

			testCompleteJSONResponse(t, respDataField.Index(i).Interface(), jsonel)
		}
	default:
		t.Fatalf("unexpected json field type (%T) in testdata", jsonData)
	}
}

func testCompleteJSONResponse(t *testing.T, respData any, jsonData map[string]any) {
	require := require.New(t)
	fs, err := jsonFields(t, respData)
	require.NoError(err)
	for _, f := range fs {
		jsonf, ok := jsonData[f.Tag]
		require.Truef(ok, "field not present in json, name: %s, tag: %s, value: %s", f.Name, f.Tag, f.Value)
		// Note: this test currently only works for non-recursive json objects
		// whose fields all take string values. Extend the test with recursion
		// and support for e.g. numerical field values as necessary.
		jsonfv, ok := jsonf.(string)
		require.Truef(ok, "field not a json string, name: %s, tag: %s, value: %s", f.Name, f.Tag, f.Value)

		require.Equal(jsonfv, f.Value, "field(%q) value mismatch, %s != %s", f.Tag, jsonfv, f.Value)
	}
}

func structFieldByName(s any, field string) (reflect.Value, error) {
	val := reflect.ValueOf(s)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("input is not a struct (pointer)")
	}

	fieldVal := val.FieldByName(field)
	if !fieldVal.IsValid() {
		return reflect.Value{}, fmt.Errorf("field %q not found", field)
	}

	return fieldVal, nil
}

type jsonField struct {
	Name  string
	Tag   string
	Value string
}

func jsonFields(t *testing.T, obj interface{}) ([]jsonField, error) {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()

	fs := make([]jsonField, 0, val.NumField())
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		name := field.Name
		jsonTag := field.Tag.Get("json")
		valueF := val.Field(i)

		value, ok := valueF.Interface().(encoding.TextMarshaler)
		if !ok && valueF.Kind() != reflect.Pointer {
			// try with a pointer... needed for Blob because MarshalText only defined on pointer
			valueF = valueF.Addr()
			value, ok = valueF.Interface().(encoding.TextMarshaler)
		}
		if !ok {
			t.Fatalf("field[%d](%q) of type %T isn't a TextMarshaler", value, i, name)
			return nil, fmt.Errorf("field[%d](%q) of type %T isn't a TextMarshaler", value, i, name)
		}

		vstr, err := value.MarshalText()
		if err != nil {
			t.Fatalf("failed to text-marshal field[%d](%q) with value %v", i, name, value)
			return nil, fmt.Errorf("failed to text-marshal field[%d](%q) with value %v", i, name, value)
		}

		fs = append(fs, jsonField{
			Name:  name,
			Tag:   jsonTag,
			Value: string(vstr),
		})
	}
	return fs, nil
}
