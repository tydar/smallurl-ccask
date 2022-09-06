package main

import (
	"fmt"

	"github.com/tydar/smallurl-ccask/ccask"
)

func main() {
	client := ccask.NewCCaskClient("29456", "localhost", 1024)

	if err := client.Connect(); err != nil {
		fmt.Printf("Connect: %v\n", err)
		return
	}

	slModel := NewShortLinkModel(client)
	if err := slModel.SetLink("abcde", "Why is it called oven when you of in"); err != nil {
		fmt.Printf("slModel.SetUrl: %v", err)
		return
	}

	sl, err := slModel.GetLink("abcde")
	if err != nil {
		fmt.Printf("slModel.GetUrl: %v", err)
	}

	fmt.Printf("Key: %s Val: %s\n", sl.Key, sl.URL)

	if err := client.Disconnect(); err != nil {
		fmt.Printf("Disconnect: %v\n", err)
	}
}
