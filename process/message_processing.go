package process

import (
	"context"
	"github.com/FatProteins/master-thesis-code/network"
	"github.com/FatProteins/master-thesis-code/network/protocol"
	"github.com/FatProteins/master-thesis-code/setup"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type Processor struct {
	messageChan  <-chan network.Message
	actionPicker *setup.ActionPicker
}

func NewProcessor(messageChan <-chan network.Message, actionPicker *setup.ActionPicker) *Processor {
	return &Processor{messageChan: messageChan, actionPicker: actionPicker}
}

func (processor *Processor) RunAsync(ctx context.Context) {
	go func() {
		select {
		case <-ctx.Done():
			return
		case message := <-processor.messageChan:
			processor.handleMessage(message)
		}
	}()
}

func (processor *Processor) handleMessage(message network.Message) {
	defer message.FreeMessage()

	var err error
	switch message.MessageType {
	case network.VOTE_REQUEST_RECEIVED:
		m := protocol.VoteRequestReceived{}
		err = anypb.UnmarshalTo(message.MessageObject, &m, proto.UnmarshalOptions{})
	case network.VOTE_RECEIVED:
		m := protocol.VoteReceived{}
		err = anypb.UnmarshalTo(message.MessageObject, &m, proto.UnmarshalOptions{})
	case network.LOG_ENTRY_REPLICATED:
		m := protocol.LogEntryReplicated{}
		err = anypb.UnmarshalTo(message.MessageObject, &m, proto.UnmarshalOptions{})
	case network.LOG_ENTRY_COMMITTED:
		m := protocol.LogEntryCommitted{}
		err = anypb.UnmarshalTo(message.MessageObject, &m, proto.UnmarshalOptions{})
	case network.LEADER_SUSPECTED:
		m := protocol.LeaderSuspected{}
		err = anypb.UnmarshalTo(message.MessageObject, &m, proto.UnmarshalOptions{})
	case network.FOLLOWER_SUSPECTED:
		m := protocol.FollowerSuspected{}
		err = anypb.UnmarshalTo(message.MessageObject, &m, proto.UnmarshalOptions{})
	}
	if err != nil {

	}

	action := processor.actionPicker.DetermineAction()
	go action.Perform()
}
