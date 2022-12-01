package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ontology-tech/Elog-go-sdk/client"
	"github.com/ontology-tech/Elog-go-sdk/utils"
)

func main() {
	client := client.NewElogClient("http://127.0.0.1:8081", "0x2099AE22B13E2228C79759795e8356C9c9989a84", "amqp://admin:kk123456@localhost:5672/", "127.0.0.1:8082")
	err := client.Register()
	if err != nil {
		panic(err)
	}
	ch1, err := client.UploadContract("eth", "", "0xa4991609c508b6D4fb7156426dB0BD49Fe298bd8", utils.ERC721)
	if err != nil {
		panic(err)
	}
	err = client.SubscribeEvents("eth", "0xa4991609c508b6D4fb7156426dB0BD49Fe298bd8", []string{"Approval", "Transfer"})
	if err != nil {
		panic(err)
	}
	go func() {
		for msg := range ch1 {
			event := &utils.Event{}
			err := json.Unmarshal(msg.Body, event)
			if err != nil {
				panic(err)
			}
			fmt.Println(event.Address, event.Height, event.Chain, event.Name)
			msg.Ack(false)
		}
	}()
	time.Sleep(10 * time.Minute)
}
