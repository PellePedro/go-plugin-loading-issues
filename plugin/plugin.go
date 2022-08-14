package main
import (
	"context"
	"skyramp/demo"
	"encoding/json"
	"fmt"
	"github.com/apache/thrift/lib/go/thrift"
	config "github.com/pellepedro/plugin/config"
)

type CartServiceHandler struct {
	AddItemResult error
	GetCartResult *demo.Cart
	EmptyCartResult error
}

func (h *CartServiceHandler) AddItem(ctx context.Context, user_id string, item *demo.CartItem) error {
	return nil
}
func (h *CartServiceHandler) GetCart(ctx context.Context, user_id string) (*demo.Cart , error) {
	return h.GetCartResult, nil
}
func (h *CartServiceHandler) EmptyCart(ctx context.Context, user_id string) error {
	return nil
}

func (h *CartServiceHandler) Configure( method string, jsonStr string) error {
	switch method {
	  case "AddItem":
		return fmt.Errorf("method [AddItem] has no return arguments")
	  case "GetCart":
		if err := json.Unmarshal([]byte(jsonStr), &h.GetCartResult ); err != nil {
			return fmt.Errorf("failed to configure mock endpoint %s", method )
		}
	  case "EmptyCart":
		return fmt.Errorf("method [EmptyCart] has no return arguments")
	}
	return nil
}

func (h *CartServiceHandler) GetProcessor() thrift.TProcessor {
	return demo.NewCartServiceProcessor(&Handler)
}

func GetHandler() config.Configure {
	return &Handler
}

var Handler = CartServiceHandler{}
