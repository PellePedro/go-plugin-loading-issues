package main

import (
	"context"
	"skyramp/demo"
	"fmt"
	"os"
	"path/filepath"
	"plugin"

	"github.com/apache/thrift/lib/go/thrift"
	config "github.com/pellepedro/plugin/config"
)

var mockA = ` {"user_id":"Skyramp",
"items": [
  {"product_id":"123","quantity":3},
  {"product_id":"456","quantity":5}
  ]
}`

type Configure interface {
	Configure(method string, json string) error
	GetProcessor() thrift.TProcessor
}

func main() {
	fpath, err := os.Getwd()
	if err != nil {
		panic("failed to find path to current working directory")
	}
	pluginPath := filepath.Join(fpath, "thrift.so")
	fmt.Printf("using plugin path %s\n", pluginPath)

	p, err := plugin.Open(pluginPath)
	if err != nil {
		fmt.Printf("failed to load plugin %v)", err)
		os.Exit(1)
	}

	h, err := p.Lookup("GetHandler")
	if err != nil {
		panic("error looking up handler")
	}
	config := h.(func() config.Configure)()

	fmt.Println("calling GetCart by thrift API")
	err = config.Configure("GetCart", mockA)
	if err != nil {
		fmt.Println(err)
	}
	processor := config.GetProcessor()

	// Start Server
	opt := NewDefaultOption()
	go func() {
		err := NewThriftServer("0.0.0.0:50061", opt, processor)
		if err != nil {
			fmt.Printf("failed to start thrift stack %#v", err)
		}
	}()
	// Start client
	client, transport, err := NewThriftClient("localhost:50061", opt)
	if err != nil {
		fmt.Printf("Failed to create client %v", err)
		os.Exit(2)
	}
	defer transport.Close()

	c := demo.NewCartServiceClient(client)
	cart, err := c.GetCart(context.Background(), "Skyramp")
	if err != nil {
		fmt.Printf("failed to get cart %v", err)
		os.Exit(2)
	}

	fmt.Println(cart)
	select {}
}
