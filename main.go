package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/ontology-tech/Elog-go-sdk/client"
	"github.com/ontology-tech/Elog-go-sdk/utils"
)

func main() {
	cli := client.NewElogClient("http://127.0.0.1:8081", "0x021143b8DAe0a232Ac90D20C78f1a5D1dE7B7dc5", "amqp://admin:kk123456@localhost:5672/", "127.0.0.1:8082")
	err := cli.Register()
	if err != nil {
		log.Panicf("Register err:%s", err.Error())
	}
	ch1, err := cli.ChaseBlock("eth", "./1.abi", "0xdAC17F958D2ee523a2206206994597C13D831ec7", utils.OTHER, 16232308, []string{"Transfer", "Approval"}, true)
	if err != nil {
		panic(err)
	}
	for body := range ch1 {
		event := &utils.Event{}
		err = json.Unmarshal(body.Body, event)
		if err != nil {
			panic(err)
		}
		fmt.Println(event.Height, event.Name)
		body.Ack(false)
	}
	
}
// 订阅事件
// result, err := cli.GetSubEvents("eth", "0xdAC17F958D2ee523a2206206994597C13D831ec7")
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Println(result)


// timestamp native erc20
	// c, err := cli.GetTimestamp("eth", 16231843)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(c)
	// t, err := cli.GetNativeToken("eth", "0xef2dd2b0f49c8175da59fda1f7fb0896a4fa251e")
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(t)
	// t1, err := cli.GetErc20Token("eth", "0xdac17f958d2ee523a2206206994597c13d831ec7", "0xdAC17F958D2ee523a2206206994597C13D831ec7")
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(t1)

	// ch, err := cli.Restart()
	// if err != nil {
	// 	panic(err.Error())
	// }
	// ch1, ok := ch["0xdAC17F958D2ee523a2206206994597C13D831ec7"]
	// if ok {
	// 	for body := range ch1 {
	// 		event := &utils.Event{}
	// 		err = json.Unmarshal(body.Body, event)
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		fmt.Println(event.Height, event.Name)
	// 		body.Ack(false)
	// 	}
	// }


	// register upload subscribe
// func main() {
// 	cli := client.NewElogClient("http://127.0.0.1:8081", "0x011143b8DAe0a232Ac90D20C78f1a5D1dE7B7dc5", "amqp://admin:kk123456@localhost:5672/", "127.0.0.1:8082")
// 	err := cli.Register()
// 	if err != nil {
// 		log.Panicf("Register err:%s", err)
// 	}
// 	ch1, err := cli.UploadContract("eth", "", "0xdAC17F958D2ee523a2206206994597C13D831ec7", utils.ERC20, true)
// 	if err != nil {
// 		log.Panicf("UploadContract err:%s", err)
// 	}
// 	err = cli.SubscribeEvents("eth", "0xdAC17F958D2ee523a2206206994597C13D831ec7", []string{"Transfer", "Approval"})
// 	if err != nil {
// 		log.Panicf("SubscribeEvents err:%s", err)
// 		return
// 	}
// 	for body := range ch1 {
// 		event := &utils.Event{}
// 		err = json.Unmarshal(body.Body, event)
// 		if err != nil {
// 			panic(err)
// 		}
// 		fmt.Println(event.Height, event.Name)
// 		body.Ack(false)
// 	}

// }