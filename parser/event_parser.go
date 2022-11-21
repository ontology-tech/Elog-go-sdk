package parser

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ontology-tech/Elog-go-sdk/utils"
)

var SysEventParser = &EventParser{}

type EventParser struct{}

func (parser *EventParser) Parser(contractAbi abi.ABI, event utils.Event) (map[string]interface{}, error) {
	abiEvent, ok := contractAbi.Events[event.Name]
	if !ok {
		return nil, utils.ErrContractNotHaveEvent
	}
	params := abiEvent.Inputs
	data, err := contractAbi.Unpack(event.Name, event.Data)
	if err != nil {
		return nil, err
	}
	result := make(map[string]interface{})
	for i := 0; i < len(event.Topics)-1; i++ {
		result[params[i].Name] = event.Topics[i+1]
	}
	for i := 0; i < len(data); i++ {
		result[params[len(event.Topics)+i-1].Name] = data[i]
	}
	return result, nil
}
