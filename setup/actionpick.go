package setup

import "github.com/FatProteins/master-thesis-code/network/protocol"

type IActionPicker interface {
	DetermineAction() FaultAction
	GetAction(actionType protocol.ActionType) FaultAction
}
