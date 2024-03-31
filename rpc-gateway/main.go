package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

func proxyRequest(ctx context.Context, r *http.Request, url string) error {
	req, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	fmt.Printf("Response: %v", resp)

	return nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: http-gateway <url1> <url2>")
		return
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		go proxyRequest(ctx, r, os.Args[1])
		go proxyRequest(ctx, r, os.Args[2])

		time.Sleep(10 * time.Second)
	})

	fmt.Println("Server started at :8888")
	http.ListenAndServe(":8888", nil)
}
