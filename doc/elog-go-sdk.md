### elog-go-sdk设计使用文档

Elog-go-sdk是针对于elog在go实现的sdk，

#### 使用步骤

创建elog访问客户端

```go
client := NewElogClient(addr, did, mqAddr) 
```

其中addr是elog对外接口，mqAddr是mq的访问接口，did是该应用的唯一标识符，最开始一般为一个用户地址，如0xaa181b97eC1989bD03061CaB7924797126B526C1，当该用户还没有注册时，did设置为空即可，如下所示

```go
client := NewElogClient("http://127.0.0.1:8081", "", "amqp://admin:kk123456@localhost:5673/") 
// 用户注册
err := client.Regioster(userAddr)
// 注册合约
mqChan, err := client.UploadContract(chain, abiPath, address, contractType)
// 注册监听事件
err := client.SubscribeEvents(address, events)  // events 为监听事件的事件名数组
//获取event
go func() {
  for msg := range mqChan {
    fmt.Println(msg.Body)
    // 当然也可以传入的数据序列化
    event := &Event{}
    err := json.Unmarshal(msg.Body, event)
    /*
    event中包含事件hash，事件块高， 事件名字，事件， chain， 合约地址，Topics, data
    其中Topic的index为0时，不再是eventhash，而是事件中indexed数据，
    例如在etherscan中topic的index 为0的为0x4c209b5fc8ad50758f13e2e1088ba56a560dff690a1c6fef26394f4c03821c4f，1为事件发起人地址0xaa181b97ec1989bd03061cab7924797126b526c1,而在event中index为0时，数据不再是eventhash，而是事件发起人地址0xaa181b97ec1989bd03061cab7924797126b526c1，如果data为地址之类的十六进制单个数据，可以直接使用，如果是十进制单个数据可以调用
HexToDec进行转化，将hex转化为int64.如果data为string，或者是多个，可以调用
SysEventParser.Parser(contractAbi, event)得到一个map，键为事件中的参数名，值当然就是参数值了
    */
  }
}
```

如果用户已经注册，且合约也已经注册，那么就可以直接生成通道·

```go
client := NewElogClient("http://127.0.0.1:8081", "did:etho:aa181b97eC1989bD03061CaB7924797126B526C1", "amqp://admin:kk123456@localhost:5673/") 
mqChans := client.Restart() // 返回的结果为map 键为用户注册的合约地址，值为mq通道，使用上面的方法从通道中拿取数据进行处理
```

```
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/orange-protocol/Elog-go-sdk/client"
	"github.com/orange-protocol/Elog-go-sdk/utils"
)

func main() {
	client := client.NewElogClient("http://127.0.0.1:8081", "", "amqp://admin:kk123456@localhost:5673/")
	err := client.Register("0xaa181b97eC1989bD03061CaB7924797126B526C1")
	if err != nil {
		panic(err)
	}
	mqChan, err := client.ChaseBlock("eth", "", "0xdAC17F958D2ee523a2206206994597C13D831ec7", utils.ERC20, 15866379, []string{"Transfer"})
	if err != nil {
		panic(err)
	}
	for msg := range mqChan {
		event := &utils.Event{}
        err := json.Unmarshal(msg.Body, event)
		if err != nil {
			log.Println(err)
		} else {
			fmt.Println(event.Address, event.Name, event.Height)
		}
	}
}
```