package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// parseFlag reads a flag argument. It assumes the flag has an argument
func parseFlag(args []string, flag string) (string, error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, flag) {
			toks := strings.Split(arg, "=")
			if len(toks) == 2 {
				return toks[1], nil
			} else if i+1 == len(args) {
				return "", fmt.Errorf("flag needs an argument: %s", flag)
			} else {
				return args[i+1], nil
			}
		}
	}
	return "", fmt.Errorf("missing flag: %s", flag)
}

func parsePathFlag(args []string, flag string) (string, error) {
	path, err := parseFlag(args, flag)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("file `%s` does not exist", path)
	}
	return path, nil
}
