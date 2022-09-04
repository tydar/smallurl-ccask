package main

import (
	"fmt"

	"github.com/tydar/smallurl-ccask/ccask"
)

func main() {
	srv := ccask.NewCCaskClient("29456", "localhost", 1024)

	if err := srv.Connect(); err != nil {
		fmt.Printf("Connect: %v\n", err)
		return
	}

	res, err := srv.GetRes([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	if err != nil {
		fmt.Printf("GetRes: %v\n", err)
		return
	}

	if err := srv.Disconnect(); err != nil {
		fmt.Printf("Disconnect: %v\n", err)
	}

	fmt.Printf("Value: %s", res.Value())
}
