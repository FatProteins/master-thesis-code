package process

import (
	"context"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/FatProteins/master-thesis-code/network"
	"github.com/FatProteins/master-thesis-code/network/protocol"
	"github.com/FatProteins/master-thesis-code/setup"
)

var logger = daLogger.NewLogger("process")

type Processor struct {
	messageChan  <-chan network.Message
	respChan     chan<- *protocol.Message
	actionPicker *setup.ActionPicker
}

func NewProcessor(messageChan <-chan network.Message, respChan chan<- *protocol.Message, actionPicker *setup.ActionPicker) *Processor {
	return &Processor{messageChan: messageChan, respChan: respChan, actionPicker: actionPicker}
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
	logger.Debug("Handling message")

	logger.Debug("Unread messages in queue: %d", len(processor.messageChan))
	action := processor.actionPicker.DetermineAction(message.LogMessage)
	logger.Info("Performing '%s' action", action.Name())
	action.Perform(message.ResetConn, processor.respChan)
	logger.Info("Done with '%s' action", action.Name())
}
