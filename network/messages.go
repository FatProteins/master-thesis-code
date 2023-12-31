package network

import (
	"errors"
	"fmt"
	"github.com/FatProteins/master-thesis-code/network/protocol"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"reflect"
)

type MessageType int

type Message struct {
	*protocol.Message
	response      *protocol.Message
	closeFunc     func(*protocol.Message)
	respondFunc   func(*protocol.Message)
	resetConnFunc func()
}

func (message *Message) FreeMessage() {
	message.closeFunc(message.Message)
	message.closeFunc(message.response)
}

func (message *Message) GetResponse() *protocol.Message {
	return message.response
}

func (message *Message) Respond() {
	message.respondFunc(message.response)
}

func (message *Message) ResetConn() {
	message.resetConnFunc()
}

const (
	HEARTBEAT             = "HEARTBEAT"
	VOTE_REQUEST_RECEIVED = "VOTE_REQUEST_RECEIVED"
	VOTE_RECEIVED         = "VOTE_RECEIVED"
	LOG_ENTRY_REPLICATED  = "LOG_ENTRY_REPLICATED"
	LOG_ENTRY_COMMITTED   = "LOG_ENTRY_COMMITTED"
	LEADER_SUSPECTED      = "LEADER_SUSPECTED"
	FOLLOWER_SUSPECTED    = "FOLLOWER_SUSPECTED"
)

func CastMessage[TargetMessageType protocol.Message](message *anypb.Any) (TargetMessageType, error) {
	output := new(TargetMessageType)
	msgBuf := protocol.Message{}
	err := anypb.UnmarshalTo(message, &msgBuf, proto.UnmarshalOptions{})
	if err != nil {
		err = errors.Join(err, fmt.Errorf("could not cast message to '%s'", reflect.TypeOf(*output).Name()))
		return *output, err
	}

	return *output, nil
}

func create() {
	x := protocol.FollowerSuspected{LeaderId: 1, FollowerId: 1}
	proto.Marshal(&x)
}
