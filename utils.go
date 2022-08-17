package sdk

import (
	"errors"
	"strings"
)

type Event struct {
	Chain   string   `json:"chain"`
	Address string   `json:"address"`
	TxHash  string   `json:"txHash"`
	Name    string   `json:"name"`
	Topics  []string `json:"topics"`
	Data    []byte   `json:"data"`
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

func Types(contractType string) bool {
	if strings.EqualFold(ERC20, contractType) ||
		strings.EqualFold(ERC721, contractType) ||
		strings.EqualFold(ERC1155, contractType) ||
		strings.EqualFold(OTHER, contractType) {
		return true
	}
	return false
}
