package network

import (
	"context"
	"fmt"
	"github.com/FatProteins/master-thesis-code/network/protocol"
	"github.com/FatProteins/master-thesis-code/util"
	"google.golang.org/protobuf/proto"
	"net"
	"os"
)

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
				_, _ = fmt.Fprintf(os.Stderr, "Failed to read message from unix socket: '%s'\n", err.Error())
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}

			fmt.Printf("Read msg of length %d\n", bytesRead)

			messageBuffer = messageBuffer[:bytesRead]
			protoMsg := networkLayer.messagePool.Get()

			err = proto.Unmarshal(messageBuffer, protoMsg)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Failed to unmarshal message: '%s'\n", err.Error())
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
				fmt.Println("Putting back msg to pool")
				networkLayer.messagePool.Put(message)
				fmt.Println("Done back msg to pool")
			}}:
			}
		}
	}()
}

func (NetworkLayer *NetworkLayer) Close() error {
	return NetworkLayer.UnixConn.Close()
}
