package main

import "os"

func main() {
	_, _ = os.Stdout.Write([]byte("hello world!\n"))
}
