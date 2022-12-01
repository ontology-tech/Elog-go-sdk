package client

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/ontology-tech/Elog-go-sdk/mq"
	"github.com/ontology-tech/Elog-go-sdk/utils"
	"github.com/spf13/cast"
	"github.com/streadway/amqp"
)

type ElogClient struct {
	addr   string // for example: "http://127.0.0.1:8081"
	wallet string //
	// mqAddr string  for example: amqp://admin:kk123456@localhost:5673/
	consumer      *mq.Consumer
	heartbeatAddr string
	cancel        context.CancelFunc
}

func NewElogClient(addr string, wallet string, mqAddr string, heartbeatAddr string) *ElogClient {
	consumer := mq.NewConsumer(mqAddr)
	return &ElogClient{
		addr:          addr,
		wallet:        wallet,
		consumer:      consumer,
		heartbeatAddr: heartbeatAddr,
	}
}

func (client *ElogClient) Register() error {
	if client.wallet == "" {
		panic("wallet is nil")
	}
	form := make(url.Values)
	form.Add("wallet", client.wallet)
	resp, err := http.PostForm(client.addr+"/register", form)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusInternalServerError {
		return utils.ErrInteralServer
	}
	if resp.StatusCode == http.StatusBadRequest {
		return errors.New(resp.Status)
	}
	did, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	client.wallet = string(did)
	// start heartbeat
	conn, err := net.Dial("tcp", client.heartbeatAddr)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	client.cancel = cancel
	go client.heartbeat(conn, ctx)
	return nil
}

func (client *ElogClient) Restart() (map[string]<-chan amqp.Delivery, error) {
	form := make(url.Values)
	form.Add("did", client.wallet)
	resp, err := http.PostForm(client.addr+"/querycontracts", form)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusInternalServerError {
		return nil, utils.ErrInteralServer
	}
	if resp.StatusCode == http.StatusBadRequest {
		return nil, errors.New(resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if len(body) == 0 {
		return nil, nil
	}
	topicsChan := make(map[string]<-chan amqp.Delivery)
	contractsInfo := make([]*utils.ContractInfo, 0)
	err = json.Unmarshal(body, &contractsInfo)
	if err != nil {
		return nil, err
	}
	for _, contractInfo := range contractsInfo {
		topic := client.wallet + contractInfo.Chain + contractInfo.Address
		topicChan, err := client.consumer.RegisterTopic(topic)
		if err != nil {
			return nil, err
		}
		topicsChan[contractInfo.Address] = topicChan
	}
	return topicsChan, nil
}

func (client *ElogClient) UploadContract(chain string, path string, address string, contractType utils.ContractType) (<-chan amqp.Delivery, error) {
	content := []byte{}
	if contractType == utils.OTHER && chain != utils.NULS {
		file, err := os.OpenFile(path, os.O_RDONLY, 0644)
		if err != nil {
			return nil, err
		}
		content, err = ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
	}
	form := make(url.Values)
	form.Add("chain", chain)
	form.Add("did", client.wallet)
	form.Add("abi", string(content))
	form.Add("type", contractType)
	form.Add("address", address)
	resp, err := http.PostForm(client.addr+"/upload", form)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusInternalServerError {
		return nil, utils.ErrInteralServer
	}
	if resp.StatusCode == http.StatusBadRequest {
		return nil, errors.New(resp.Status)
	}
	if resp.StatusCode == http.StatusOK {
		topic := client.wallet + chain + address
		topicChan, err := client.consumer.RegisterTopic(topic)
		if err != nil {
			return nil, utils.ErrRegisterTopic
		}
		return topicChan, nil
	}
	return nil, nil
}

func (client *ElogClient) ChaseBlock(chain string, path string,
	address string, contractType utils.ContractType,
	startBlock uint64, eventsName []string) (<-chan amqp.Delivery, error) {
	content := []byte{}
	if contractType == utils.OTHER && chain != utils.NULS {
		file, err := os.OpenFile(path, os.O_RDONLY, 0644)
		if err != nil {
			return nil, err
		}
		content, err = ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
	}
	form := make(url.Values)
	form.Add("chain", chain)
	form.Add("did", client.wallet)
	form.Add("abi", string(content))
	form.Add("type", contractType)
	form.Add("address", address)
	form.Add("startBlock", cast.ToString(startBlock))
	for _, name := range eventsName {
		form.Add("names", name)
	}
	resp, err := http.PostForm(client.addr+"/chase", form)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusInternalServerError {
		return nil, utils.ErrInteralServer
	}
	if resp.StatusCode == http.StatusBadRequest {
		return nil, errors.New(resp.Status)
	}
	if resp.StatusCode == http.StatusOK {
		topic := client.wallet + chain + address
		topicChan, err := client.consumer.RegisterTopic(topic)
		if err != nil {
			return nil, utils.ErrRegisterTopic
		}
		return topicChan, nil
	}
	return nil, nil
}

func (client *ElogClient) SubscribeEvents(chain string, address string, names []string) error {
	form := make(url.Values)
	form.Add("did", client.wallet)
	form.Add("chain", chain)
	form.Add("address", address)
	for _, name := range names {
		form.Add("names", name)
	}
	resp, err := http.PostForm(client.addr+"/subscribe", form)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusInternalServerError {
		return utils.ErrInteralServer
	}
	if resp.StatusCode == http.StatusBadRequest {
		return errors.New(resp.Status)
	}
	return nil
}

func (client *ElogClient) RemoveContract(chain string, addr string) error {
	form := make(url.Values)
	form.Add("did", client.wallet)
	form.Add("chain", chain)
	form.Add("address", addr)
	resp, err := http.PostForm(client.addr+"/remove", form)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusInternalServerError {
		return utils.ErrInteralServer
	}
	if resp.StatusCode == http.StatusBadRequest {
		return errors.New(resp.Status)
	}
	return nil

}

func (client *ElogClient) GetTimestamp(chain string, height int64) (int64, error) {
	form := make(url.Values)
	form.Add("chain", chain)
	form.Add("height", cast.ToString(height))
	resp, err := http.PostForm(client.addr+"/getTime", form)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode == http.StatusInternalServerError {
		return 0, utils.ErrInteralServer
	}
	if resp.StatusCode == http.StatusBadRequest {
		return 0, errors.New(resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	return cast.ToInt64(string(body)), nil
}

func (client *ElogClient) UnSubscribeEvents(chain string, addr string, names []string) error {
	form := make(url.Values)
	form.Add("did", client.wallet)
	form.Add("chain", chain)
	form.Add("address", addr)
	for _, name := range names {
		form.Add("names", name)
	}
	resp, err := http.PostForm(client.addr+"/unsubsribe", form)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusInternalServerError {
		return utils.ErrInteralServer
	}
	if resp.StatusCode == http.StatusBadRequest {
		return errors.New(resp.Status)
	}
	return nil
}

func (client *ElogClient) Close() {
	client.cancel()
}

func (client *ElogClient) heartbeat(conn net.Conn, ctx context.Context) {
	ticker := time.NewTicker(time.Duration(10) * time.Second)
	msg := client.wallet + "\n"
	writer := bufio.NewWriter(conn)
Loop:
	for {
		select {
		case <-ticker.C:
			_, err := writer.WriteString(msg)
			if err != nil {
				log.Println("conn WriteString fail", err)
				break
			} 
			err = writer.Flush()
			if err != nil {
				log.Println("writer flush fail", err.Error())
			}
		case <-ctx.Done():
			log.Println("conn close")
			break Loop
		}
	}
}

