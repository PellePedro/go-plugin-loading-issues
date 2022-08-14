package main

import (
	"context"
	"demo"
	"testing"

	// "github.com/apache/thrift/lib/go/thrift"
	// config "github.com/pellepedro/plugin/config"
	"github.com/stretchr/testify/assert"
)

func TestConfigureMock(t *testing.T) {
	handler := GetHandler()

	// Set mock response for endpoint GetCart
	mockA := ` {"user_id":"Skyramp",
                  "items": [
					{"product_id":"123","quantity":3},
					{"product_id":"456","quantity":5}
					]
				}`
	handler.Configure("GetCart", mockA)
	processor := handler.GetProcessor()

	// Start Server
	opt := NewDefaultOption()
	err := NewThriftServer("0.0.0.0:50051", opt, processor)
	assert.Nil(t, err)

	// Start client
	client, transport, err := NewThriftClient("localhost:50051", opt)
	if err != nil {
		panic("failed to create client")
	}
	defer transport.Close()

	c := demo.NewCartServiceClient(client)
	cart, err := c.GetCart(context.Background(), "Skyramp")
	assert.Nil(t, err)
	assert.Equal(t, "Skyramp", cart.UserID)
	assert.Equal(t, 2, len(cart.Items))
	assert.Equal(t, "123", cart.Items[0].ProductID)
	assert.Equal(t, int32(3), cart.Items[1].Quantity)
	assert.Equal(t, "456", cart.Items[1].ProductID)
	assert.Equal(t, int32(5), cart.Items[1].Quantity)

	// Update handler with new mock data
	mockB := `{"user_id":"Letsramp",
                  "items": [
					{"product_id":"789","quantity":2}
					]
				  }`
	handler.Configure("GetCart", mockB)
	cart, err = c.GetCart(context.Background(), "Skyramp")
	assert.Nil(t, err)
	assert.Equal(t, "Letsramp", cart.UserID)
	assert.Equal(t, 1, len(cart.Items))
	assert.Equal(t, "789", cart.Items[0].ProductID)
	assert.Equal(t, int32(2), cart.Items[0].Quantity)
}
