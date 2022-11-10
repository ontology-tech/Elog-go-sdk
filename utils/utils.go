package utils

import (
	"errors"
	"strconv"
	"strings"
)

type Event struct {
	Chain   string   `json:"chain"`
	Address string   `json:"address"`
	TxHash  string   `json:"txHash"`
	Name    string   `json:"name"`
	Topics  []string `json:"topics"`
	Data    []byte   `json:"data"`
	Height  int64   `json:"height"`
	BlockTime uint64 `json:"block_time"`
}

var (
	ErrAddressFormatter       = errors.New("address format error")
	ErrInteralServer          = errors.New("internal server error")
	ErrNotSupportContractType = errors.New("elog doesn't support the contract type")
	ErrTopicHasRegistered     = errors.New("topic has been subscribed")
	ErrRegisterTopic          = errors.New("register topic fail")
	ErrContractNotHaveEvent   = errors.New("the contract doesn't have the event")
)

type ContractType = string

const (
	ERC20   ContractType = "ERC20"
	ERC721  ContractType = "ERC721"
	ERC1155 ContractType = "ERC1155"
	OTHER   ContractType = "OTHER"
)

type ContractInfo struct {
	Address string `json:"address"`
	Chain   string `json:"chain"`
}

func Types(contractType string) bool {
	if strings.EqualFold(ERC20, contractType) ||
		strings.EqualFold(ERC721, contractType) ||
		strings.EqualFold(ERC1155, contractType) ||
		strings.EqualFold(OTHER, contractType) {
		return true
	}
	return false
}

func HexToDec(num string) int64 {
    val := num[2:]
    
    n, err := strconv.ParseInt(val, 16, 32)
    if err != nil {
        panic(err)
    }
   return n
}
