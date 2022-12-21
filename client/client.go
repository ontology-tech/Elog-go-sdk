package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	client        *http.Client
}

func NewElogClient(addr string, wallet string, mqAddr string, heartbeatAddr string) *ElogClient {
	consumer := mq.NewConsumer(mqAddr)
	return &ElogClient{
		addr:          addr,
		wallet:        wallet,
		consumer:      consumer,
		heartbeatAddr: heartbeatAddr,
		client:        &http.Client{Timeout: 5 * time.Second},
	}
}

func (client *ElogClient) Close() {
	client.cancel()
}

func (elogClient *ElogClient) Register() error {
	if elogClient.wallet == "" {
		return errors.New("wallet is nil")
	}
	form := make(url.Values)
	form.Add("wallet", elogClient.wallet)
	url := elogClient.addr + "/register"
	resp, err := elogClient.client.PostForm(url, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get data from %s status error: %d", url, resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	elogResp := &utils.ElogResponse{}
	err = json.Unmarshal(data, elogResp)
	if err != nil {
		return err
	}
	if elogResp.Error != "" {
		return errors.New(elogResp.Error)
	}
	elogClient.wallet = elogResp.Result.(string)
	// start heartbeat
	conn, err := net.Dial("tcp", elogClient.heartbeatAddr)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	elogClient.cancel = cancel
	go elogClient.heartbeat(conn, ctx)
	return nil
}

func (elogClient *ElogClient) UploadContract(chain string, path string, address string, contractType utils.ContractType, isEvm bool) (<-chan amqp.Delivery, error) {
	url := elogClient.addr + "/upload"
	form := make(map[string]string)
	form["did"] = elogClient.wallet
	form["chain"] = chain
	form["address"] = address
	form["type"] = contractType
	neeUpload := false
	if contractType == utils.OTHER && isEvm {
		neeUpload = true
	}
	buf, contentType, err := elogClient.createMultipartReq(path, form, neeUpload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()
	req.Header.Set("Content-Type", contentType)
	resp, err := elogClient.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get data from %s status error: %d", url, resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	elogResp := &utils.ElogResponse{}
	err = json.Unmarshal(data, elogResp)
	if err != nil {
		return nil, err
	}
	if elogResp.Error != "" {
		return nil, errors.New(elogResp.Error)
	}
	topic := elogClient.wallet + chain + address
	topicChan, err := elogClient.consumer.RegisterTopic(topic)
	if err != nil {
		return nil, utils.ErrRegisterTopic
	}
	return topicChan, nil
}

func (elogClient *ElogClient) SubscribeEvents(chain string, address string, names []string) error {
	form := make(url.Values)
	form.Add("did", elogClient.wallet)
	form.Add("chain", chain)
	form.Add("address", address)
	for _, name := range names {
		form.Add("names", name)
	}
	url := elogClient.addr + "/subscribe"
	resp, err := elogClient.client.PostForm(url, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get data from %s status error: %d", url, resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	elogResp := &utils.ElogResponse{}
	err = json.Unmarshal(data, elogResp)
	if err != nil {
		return err
	}
	if elogResp.Error != "" {
		return errors.New(elogResp.Error)
	}
	return nil
}

func (elogClient *ElogClient) ChaseBlock(chain string, path string, address string, contractType utils.ContractType,
	startBlock uint64, eventsName []string, isEvm bool) (<-chan amqp.Delivery, error) {
	url := elogClient.addr + "/chase"
	form := make(map[string]string)
	form["did"] = elogClient.wallet
	form["chain"] = chain
	form["address"] = address
	form["type"] = contractType
	form["startBlock"] = cast.ToString(startBlock)
	form["names"] = strings.Join(eventsName, ",")
	neeUpload := false
	if contractType == utils.OTHER && isEvm {
		neeUpload = true
	}
	buf, contentType, err := elogClient.createMultipartReq(path, form, neeUpload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()
	req.Header.Set("Content-Type", contentType)
	resp, err := elogClient.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get data from %s status error: %d", url, resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	elogResp := &utils.ElogResponse{}
	err = json.Unmarshal(data, elogResp)
	if err != nil {
		return nil, err
	}
	if elogResp.Error != "" {
		return nil, errors.New(elogResp.Error)
	}
	topic := elogClient.wallet + chain + address
	topicChan, err := elogClient.consumer.RegisterTopic(topic)
	if err != nil {
		return nil, utils.ErrRegisterTopic
	}
	return topicChan, nil
}

func (elogClient *ElogClient) RemoveContract(chain string, addr string) error {
	form := make(url.Values)
	form.Add("did", elogClient.wallet)
	form.Add("chain", chain)
	form.Add("address", addr)
	url := elogClient.addr + "/remove"
	resp, err := elogClient.client.PostForm(url, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get data from %s status error: %d", url, resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	elogResp := &utils.ElogResponse{}
	err = json.Unmarshal(data, elogResp)
	if err != nil {
		return err
	}
	if elogResp.Error != "" {
		return errors.New(elogResp.Error)
	}
	return nil
}

func (elogClient *ElogClient) Restart() (map[string]<-chan amqp.Delivery, error) {
	form := make(url.Values)
	form.Add("did", elogClient.wallet)
	url := elogClient.addr + "/querycontracts"
	resp, err := elogClient.client.PostForm(url, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get data from %s status error: %d", url, resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	elogResp := &utils.ElogResponse{}
	err = json.Unmarshal(data, elogResp)
	if err != nil {
		return nil, err
	}
	if elogResp.Error != "" {
		return nil, errors.New(elogResp.Error)
	}
	result := elogResp.Result.(string)
	var contractInfos []*utils.ContractInfo
	err = json.Unmarshal([]byte(result), &contractInfos)
	if err != nil {
		return nil, err
	}
	fmt.Printf("%s subscribe %d contracts", elogClient.wallet, len(contractInfos))
	topicsChan := make(map[string]<-chan amqp.Delivery)
	for _, contractInfo := range contractInfos {
		topic := elogClient.wallet + contractInfo.Chain + contractInfo.Address
		topicChan, err := elogClient.consumer.RegisterTopic(topic)
		if err != nil {
			return nil, err
		}
		topicsChan[contractInfo.Address] = topicChan
	}
	return topicsChan, nil
}

func (elogClient *ElogClient) GetSubEvents(chain string, contract string) ([]string, error) {
	form := make(url.Values)
	form.Add("did", elogClient.wallet)
	form.Add("chain", chain)
	form.Add("address", contract)
	url := elogClient.addr + "/queryevents"
	resp, err := elogClient.client.PostForm(url, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get data from %s status error: %d", url, resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	elogResp := &utils.ElogResponse{}
	err = json.Unmarshal(data, elogResp)
	if err != nil {
		return nil, err
	}
	if elogResp.Error != "" {
		return nil, errors.New(elogResp.Error)
	}
	result := elogResp.Result.(string)
	var events []string
	err = json.Unmarshal([]byte(result), &events)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (elogClient *ElogClient) GetTimestamp(chain string, height int64) (uint64, error) {
	form := make(url.Values)
	form.Add("chain", chain)
	form.Add("height", cast.ToString(height))
	url := elogClient.addr+"/getTime"
	resp, err := elogClient.client.PostForm(url, form)
	if err != nil {
		return 0,  err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("get data from %s status error: %d", url, resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	elogResp := &utils.ElogResponse{}
	err = json.Unmarshal(data, elogResp)
	if err != nil {
		return 0, err
	}
	if elogResp.Error != "" {
		return 0, errors.New(elogResp.Error)
	}
	
	return cast.ToUint64(elogResp.Result.(string)), nil
}

func (elogClient *ElogClient) GetNativeToken(chain string, address string) (uint64, error) {
	form := make(url.Values)
	form.Add("chain", chain)
	form.Add("address", address)
	url := elogClient.addr + "/getNativeToken"
	resp, err := elogClient.client.PostForm(url, form)
	if err != nil {
		return 0,  err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("get data from %s status error: %d", url, resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	elogResp := &utils.ElogResponse{}
	err = json.Unmarshal(data, elogResp)
	if err != nil {
		return 0, err
	}
	if elogResp.Error != "" {
		return 0, errors.New(elogResp.Error)
	}
	return cast.ToUint64(elogResp.Result.(string)), nil
}

func (elogClient *ElogClient) GetErc20Token(chain string, wallet string, contract string) (uint64, error) {
	form := make(url.Values)
	form.Add("chain", chain)
	form.Add("wallet", wallet)
	form.Add("contract", contract)
	url := elogClient.addr + "/getErc20Token"
	resp, err := elogClient.client.PostForm(url, form)
	if err != nil {
		return 0,  err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("get data from %s status error: %d", url, resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	elogResp := &utils.ElogResponse{}
	err = json.Unmarshal(data, elogResp)
	if err != nil {
		return 0, err
	}
	if elogResp.Error != "" {
		return 0, errors.New(elogResp.Error)
	}
	return cast.ToUint64(elogResp.Result.(string)), nil
}

func (client *ElogClient) createMultipartReq(path string, columns map[string]string, needUpload bool) (*bytes.Buffer, string, error) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	defer writer.Close()
	for key, value := range columns {
		err := writer.WriteField(key, value)
		if err != nil {
			return nil, "", err
		}
	}
	if needUpload {
		file, err := os.Open(path)
		if err != nil {
			return nil, "", err
		}
		defer file.Close()
		fileWriter, err := writer.CreateFormFile("file", path)
		if err != nil {
			return nil, "", err
		}
		_, err = io.Copy(fileWriter, file)
		if err != nil {
			return nil, "", err
		}
	}
	return buf, writer.FormDataContentType(), nil
}

func (client *ElogClient) heartbeat(conn net.Conn, ctx context.Context) {
	ticker := time.NewTicker(time.Duration(30) * time.Second)
	msg := client.wallet + "\n"
	writer := bufio.NewWriter(conn)
Loop:
	for {
		select {
		case <-ticker.C:
			_, err := writer.WriteString(msg)
			if err != nil {
				fmt.Println("conn WriteString fail", err)
				break
			}
			err = writer.Flush()
			if err != nil {
				fmt.Println("writer flush fail", err.Error())
			}
		case <-ctx.Done():
			fmt.Println("conn close")
			break Loop
		}
	}
}
