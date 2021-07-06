package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

func main() {
	var (
		region  string = "us-east-1"
		secname string
		verbose bool
	)
	flag.StringVar(&region, "region", region, "AWS region")
	flag.StringVar(&secname, "name", secname, "secret name")
	flag.BoolVar(&verbose, "verbose", verbose, "print error output (if any)")
	flag.Parse()

	if secname == "" {
		if verbose {
			fmt.Println("please provide the secret's name")
		}
		os.Exit(1)
	}

	svc := secretsmanager.New(session.New(), aws.NewConfig().WithRegion(region))
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secname),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}
	result, err := svc.GetSecretValue(input)
	if err != nil {
		if verbose {
			fmt.Printf("error retrieving the secret value: %v\n", err)
		}
		os.Exit(1)
	}
	kv := map[string]string{}
	if err := json.Unmarshal([]byte(*result.SecretString), &kv); err != nil {
		if verbose {
			fmt.Printf("error parsing the secret's json contents: %v\n", err)
		}
		os.Exit(1)
	}
	for k, v := range kv {
		fmt.Printf("%s=%s\n", k, v)
	}
}
