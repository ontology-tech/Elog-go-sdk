package client

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/orange-protocol/Elog-go-sdk/mq"
	"github.com/orange-protocol/Elog-go-sdk/utils"
	"github.com/spf13/cast"
	"github.com/streadway/amqp"
)

type ElogClient struct {
	addr string // for example: "http://127.0.0.1:8081"
	did  string //
	// mqAddr string  for example: amqp://admin:kk123456@localhost:5673/
	consumer *mq.Consumer
}

func NewElogClient(addr string, did string, mqAddr string) *ElogClient {
	consumer := mq.NewConsumer(mqAddr)
	return &ElogClient{
		addr:     addr,
		did:      did,
		consumer: consumer,
	}
}

func (client *ElogClient) Register(wallet string) error {
	form := make(url.Values)
	form.Add("wallet", wallet)
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
	client.did = string(did)
	return nil
}

func (client *ElogClient) Restart() (map[string]<-chan amqp.Delivery, error) {
	form := make(url.Values)
	form.Add("did", client.did)
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
		topic := client.did + contractInfo.Chain + contractInfo.Address
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
	form.Add("did", client.did)
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
		topic := client.did + chain + address
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
	form.Add("did", client.did)
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
		topic := client.did + chain + address
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
	form.Add("did", client.did)
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
	form.Add("did", client.did)
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

func (client *ElogClient) UnSubscribeEvents(chain string, addr string, names []string) error {
	form := make(url.Values)
	form.Add("did", client.did)
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
