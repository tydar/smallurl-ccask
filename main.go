package main

import (
	"fmt"
	"net/http"

	"github.com/tydar/smallurl-ccask/ccask"
)

func main() {
	client := ccask.NewCCaskClient("29456", "localhost", 1024)

	if err := client.Connect(); err != nil {
		fmt.Printf("Connect: %v\n", err)
		return
	}

	defer client.Disconnect()

	env := NewEnv(client)
	if err := env.AddTemplate("set", "templates/base.html", "templates/short.html"); err != nil {
		fmt.Printf("AddTemplate: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", env.SetURLHandler)
	mux.HandleFunc("/q/", env.GetURLHandler)

	http.ListenAndServe(":8080", mux)
}
