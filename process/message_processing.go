package process

import (
	"context"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/FatProteins/master-thesis-code/network"
	"github.com/FatProteins/master-thesis-code/setup"
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

	logger.Debug("Unread messages in queue: %d", len(processor.messageChan))
	action := processor.actionPicker.DetermineAction()
	//action := processor.actionPicker.GetAction(message.ActionType)
	logger.Info("Performing '%s' action", action.Name())
	action.Perform(message.ResetConn)
	logger.Info("Done with '%s' action", action.Name())
	//response := message.GetResponse()
	//err := action.GenerateResponse(response)
	//if err != nil {
	//	logger.ErrorErr(err, "Failed to generate DA response. Sending default response instead.")
	//	response.MessageType = protocol.MessageType_DA_RESPONSE
	//}

	//message.Respond()
}
