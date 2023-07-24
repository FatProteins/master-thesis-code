package network

import (
	"context"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/FatProteins/master-thesis-code/network/protocol"
	"github.com/FatProteins/master-thesis-code/util"
	"google.golang.org/protobuf/proto"
	"net"
)

var logger = daLogger.NewLogger("network")

type NetworkLayer struct {
	*net.UnixConn
	messagePool *util.Pool[protocol.Message]
	handleChan  chan<- Message
}

func NewNetworkLayer(localAddr *net.UnixAddr, handleChan chan<- Message) (*NetworkLayer, error) {
	connection, err := net.ListenUnixgram("unixgram", localAddr)
	if err != nil {
		return nil, err
	}

	return &NetworkLayer{UnixConn: connection, messagePool: util.NewPool[protocol.Message](), handleChan: handleChan}, nil
}

func (networkLayer *NetworkLayer) RunAsync(ctx context.Context) {
	go func() {
		messageBuffer := make([]byte, 4096*10)
		for {
			bytesRead, _, err := networkLayer.ReadFromUnix(messageBuffer)
			if err != nil {
				logger.ErrorErr(err, "Failed to read message from unix socket")
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}

			logger.Info("Read msg of length %d", bytesRead)

			messageBuffer = messageBuffer[:bytesRead]
			protoMsg := networkLayer.messagePool.Get()

			err = proto.Unmarshal(messageBuffer, protoMsg)
			if err != nil {
				logger.ErrorErr(err, "Failed to unmarshal message")
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}

			select {
			case <-ctx.Done():
				return
			case networkLayer.handleChan <- Message{protoMsg, func(message *protocol.Message) {
				logger.Info("Putting back msg to pool")
				message.Reset()
				networkLayer.messagePool.Put(message)
				logger.Info("Done back msg to pool")
			}}:
			}
		}
	}()
}

func (NetworkLayer *NetworkLayer) Close() error {
	return NetworkLayer.UnixConn.Close()
}
