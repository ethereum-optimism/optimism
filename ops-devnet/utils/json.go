package utils

import (
 	"encoding/json"
 	"io/ioutil"
)

// WriteJson writes the [target] to [path] as JSON.
func WriteJson(path string, target interface{}) error {
	file, _ := json.MarshalIndent(target, "", " ")
	return ioutil.WriteFile(path, file, 0644)
}
