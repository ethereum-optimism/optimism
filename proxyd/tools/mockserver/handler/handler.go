package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/proxyd"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type MethodTemplate struct {
	Method   string `yaml:"method"`
	Block    string `yaml:"block"`
	Response string `yaml:"response"`
}

type MockedHandler struct {
	Overrides    []*MethodTemplate
	Autoload     bool
	AutoloadFile string
}

func (mh *MockedHandler) Serve(port int) error {
	r := mux.NewRouter()
	r.HandleFunc("/", mh.Handler)
	http.Handle("/", r)
	fmt.Printf("starting server up on :%d serving MockedResponsesFile %s\n", port, mh.AutoloadFile)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		return err
	}
	return nil
}

func (mh *MockedHandler) Handler(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("error reading request: %v\n", err)
	}

	var template []*MethodTemplate
	if mh.Autoload {
		template = append(template, mh.LoadFromFile(mh.AutoloadFile)...)
	}
	if mh.Overrides != nil {
		template = append(template, mh.Overrides...)
	}

	batched := proxyd.IsBatch(body)
	var requests []map[string]interface{}
	if batched {
		err = json.Unmarshal(body, &requests)
		if err != nil {
			fmt.Printf("error reading request: %v\n", err)
		}
	} else {
		var j map[string]interface{}
		err = json.Unmarshal(body, &j)
		if err != nil {
			fmt.Printf("error reading request: %v\n", err)
		}
		requests = append(requests, j)
	}

	var responses []string
	for _, r := range requests {
		method := r["method"]
		block := ""
		if method == "eth_getBlockByNumber" {
			block = (r["params"].([]interface{})[0]).(string)
		}

		var selectedResponse string
		for _, r := range template {
			if r.Method == method && r.Block == block {
				selectedResponse = r.Response
			}
		}
		if selectedResponse != "" {
			var rpcRes proxyd.RPCRes
			err = json.Unmarshal([]byte(selectedResponse), &rpcRes)
			idJson, _ := json.Marshal(r["id"])
			rpcRes.ID = idJson
			res, _ := json.Marshal(rpcRes)
			responses = append(responses, string(res))
		}
	}

	resBody := ""
	if batched {
		resBody = "[" + strings.Join(responses, ",") + "]"
	} else if len(responses) > 0 {
		resBody = responses[0]
	}

	_, err = fmt.Fprint(w, resBody)
	if err != nil {
		fmt.Printf("error writing response: %v\n", err)
	}
}

func (mh *MockedHandler) LoadFromFile(file string) []*MethodTemplate {
	contents, err := os.ReadFile(file)
	if err != nil {
		fmt.Printf("error reading MockedResponsesFile: %v\n", err)
	}
	var template []*MethodTemplate
	err = yaml.Unmarshal(contents, &template)
	if err != nil {
		fmt.Printf("error reading MockedResponsesFile: %v\n", err)
	}
	return template
}

func (mh *MockedHandler) AddOverride(template *MethodTemplate) {
	mh.Overrides = append(mh.Overrides, template)
}

func (mh *MockedHandler) ResetOverrides() {
	mh.Overrides = make([]*MethodTemplate, 0)
}
