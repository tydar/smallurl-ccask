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

	res, err := srv.SetRes([]byte{0x41, 0x42, 0x43, 0x44, 0x45}, []byte("The quick brown fox jumped over the lazy dog."))
	if err != nil {
		fmt.Printf("SetRes: %v\n", err)
		return
	}

	fmt.Printf("Set Value: %s\n", res.Value())

	res, err = srv.GetRes([]byte{0x41, 0x42, 0x43, 0x44, 0x45})
	if err != nil {
		fmt.Printf("GetRes: %v\n", err)
		return
	}

	fmt.Printf("Get Value: %s\n", res.Value())

	if err := srv.Disconnect(); err != nil {
		fmt.Printf("Disconnect: %v\n", err)
	}
}
