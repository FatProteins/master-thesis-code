package network

import (
	"context"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/FatProteins/master-thesis-code/network/protocol"
	"github.com/FatProteins/master-thesis-code/util"
	"google.golang.org/protobuf/proto"
	"net"
	"os"
	"sync/atomic"
)

var logger = daLogger.NewLogger("network")

type NetworkLayer struct {
	*net.UnixConn
	localAddr      *net.UnixAddr
	messagePool    *util.Pool[protocol.Message]
	handleChan     chan<- Message
	respChan       <-chan Message
	unixSocketPath string
	resetConn      atomic.Bool
}

func NewNetworkLayer(handleChan chan<- Message, respChan <-chan Message, localAddr *net.UnixAddr, unixSocketPath string) (*NetworkLayer, error) {

	//connection, err := net.DialUnix("unixgram", nil, remoteAddr)
	//if err != nil {
	//	return nil, err
	//}

	resetConn := atomic.Bool{}
	resetConn.Store(true)
	return &NetworkLayer{localAddr: localAddr, messagePool: util.NewPool[protocol.Message](), handleChan: handleChan, respChan: respChan, unixSocketPath: unixSocketPath, resetConn: resetConn}, nil
}

func (networkLayer *NetworkLayer) ResetConn() {
	if networkLayer.UnixConn != nil {
		networkLayer.UnixConn.Close()
		err := os.Remove(networkLayer.unixSocketPath)
		if err != nil {
			panic(err)
		}
	}

	listener, err := net.ListenUnix("unix", networkLayer.localAddr)
	if err != nil {
		panic("failed to listen to unix socket")
	}

	connection, err := listener.AcceptUnix()
	if err != nil {
		panic("failed to connect unix socket")
	}
	networkLayer.UnixConn = connection
}

func (networkLayer *NetworkLayer) SetResetConn(reset bool) {
	networkLayer.resetConn.Store(reset)
}

func (networkLayer *NetworkLayer) RunAsync(ctx context.Context) {
	go func() {
		messageBuffer := make([]byte, 4096*10)
		for {
			if networkLayer.resetConn.Load() {
				networkLayer.ResetConn()
				networkLayer.resetConn.Store(false)
			}
			bytesRead, _, err := networkLayer.ReadFromUnix(messageBuffer)
			if err != nil {
				//logger.ErrorErr(err, "Failed to read message from unix socket")
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}

			logger.Debug("Read msg of length %d", bytesRead)

			messageBuffer = messageBuffer[:bytesRead]
			protoMsg := networkLayer.messagePool.Get()

			err = proto.Unmarshal(messageBuffer, protoMsg)
			if err != nil {
				//logger.ErrorErr(err, "Failed to unmarshal message")
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
			case networkLayer.handleChan <- Message{
				Message:  protoMsg,
				response: networkLayer.messagePool.Get(),
				closeFunc: func(message *protocol.Message) {
					logger.Debug("Putting back msg to pool")
					message.Reset()
					networkLayer.messagePool.Put(message)
					logger.Debug("Done back msg to pool")
				},
				respondFunc: func(response *protocol.Message) {
					logger.Debug("Responding with response '%s'", response.String())
					respBytes, err := proto.Marshal(response)
					if err != nil {
						logger.ErrorErr(err, "Failed to marshal DA response to bytes")
						return
					}

					bytesWritten, err := networkLayer.Write(respBytes)
					if err != nil {
						logger.ErrorErr(err, "Failed to send DA response")
						return
					}

					logger.Debug("Sent DA response with length %d", bytesWritten)
				},
				resetConnFunc: func() {
					networkLayer.resetConn.Store(true)
				},
			}:
			}
		}
	}()
	//
	//go func() {
	//	for {
	//		select {
	//		case <-ctx.Done():
	//			return
	//		case response := <-networkLayer.respChan:
	//			respBytes, err := proto.Marshal(response)
	//			if err != nil {
	//				logger.ErrorErr(err, "Failed to marshal DA response to bytes")
	//				response.FreeMessage()
	//				continue
	//			}
	//
	//			bytesWritten, err := networkLayer.Write(respBytes)
	//			if err != nil {
	//				logger.ErrorErr(err, "Failed to send DA response")
	//				response.FreeMessage()
	//				continue
	//			}
	//
	//			logger.Debug("Sent DA response with length %d", bytesWritten)
	//			response.FreeMessage()
	//		}
	//	}
	//}()
}

func (networkLayer *NetworkLayer) Close() error {
	return networkLayer.UnixConn.Close()
}
