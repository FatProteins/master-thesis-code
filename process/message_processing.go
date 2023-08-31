package process

import (
	"context"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/FatProteins/master-thesis-code/network"
	"github.com/FatProteins/master-thesis-code/network/protocol"
	"github.com/FatProteins/master-thesis-code/setup"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var logger = daLogger.NewLogger("process")

type Processor struct {
	messageChan  <-chan network.Message
	respChan     chan<- network.Message
	actionPicker *setup.ActionPicker
}

func NewProcessor(messageChan <-chan network.Message, respChan chan<- network.Message, actionPicker *setup.ActionPicker) *Processor {
	return &Processor{messageChan: messageChan, actionPicker: actionPicker}
}

func (processor *Processor) RunAsync(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case message := <-processor.messageChan:
				processor.handleMessage(message)
			}
		}
	}()
}

func (processor *Processor) handleMessage(message network.Message) {
	defer message.FreeMessage()
	logger.Debug("Handling message")

	var err error
	switch message.MessageType {
	case protocol.MessageType_VOTE_REQUEST_RECEIVED:
		m := protocol.VoteRequestReceived{}
		err = anypb.UnmarshalTo(message.MessageObject, &m, proto.UnmarshalOptions{})
	case protocol.MessageType_VOTE_RECEIVED:
		m := protocol.VoteReceived{}
		err = anypb.UnmarshalTo(message.MessageObject, &m, proto.UnmarshalOptions{})
	case protocol.MessageType_LOG_ENTRY_REPLICATED:
		m := protocol.LogEntryReplicated{}
		err = anypb.UnmarshalTo(message.MessageObject, &m, proto.UnmarshalOptions{})
	case protocol.MessageType_LOG_ENTRY_COMMITTED:
		m := protocol.LogEntryCommitted{}
		err = anypb.UnmarshalTo(message.MessageObject, &m, proto.UnmarshalOptions{})
	case protocol.MessageType_LEADER_SUSPECTED:
		m := protocol.LeaderSuspected{}
		err = anypb.UnmarshalTo(message.MessageObject, &m, proto.UnmarshalOptions{})
	case protocol.MessageType_FOLLOWER_SUSPECTED:
		m := protocol.FollowerSuspected{}
		err = anypb.UnmarshalTo(message.MessageObject, &m, proto.UnmarshalOptions{})
	}
	if err != nil {

	}

	logger.Debug("Unread messages in queue: %d", len(processor.messageChan))
	//action := processor.actionPicker.DetermineAction()
	action := processor.actionPicker.GetAction(message.ActionType)
	logger.Info("Performing '%s' action", action.Name())
	action.Perform()
	logger.Info("Done with '%s' action", action.Name())
	response := message.GetResponse()
	err = action.GenerateResponse(response)
	if err != nil {
		logger.ErrorErr(err, "Failed to generate DA response. Sending default response instead.")
		response.MessageType = protocol.MessageType_DA_RESPONSE
	}

	message.Respond()
}
