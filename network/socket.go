package network

import (
	"context"
	"encoding/json"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/FatProteins/master-thesis-code/network/protocol"
	"github.com/FatProteins/master-thesis-code/util"
	"github.com/google/uuid"
	"net"
	"os"
	"sync"
	"sync/atomic"
)

var logger = daLogger.NewLogger("network")

type NetworkLayer struct {
	*net.UnixConn
	localAddr      *net.UnixAddr
	messagePool    *util.Pool[protocol.Message]
	handleChan     chan<- Message
	respChan       chan *protocol.Message
	unixSocketPath string
	resetConn      *atomic.Bool
	logCallbacks   sync.Map
}

func NewNetworkLayer(handleChan chan<- Message, respChan chan *protocol.Message, localAddr *net.UnixAddr, unixSocketPath string) (*NetworkLayer, error) {

	//connection, err := net.DialUnix("unixgram", nil, remoteAddr)
	//if err != nil {
	//	return nil, err
	//}

	resetConn := atomic.Bool{}
	resetConn.Store(true)
	return &NetworkLayer{
		localAddr:      localAddr,
		messagePool:    util.NewPool[protocol.Message](),
		handleChan:     handleChan,
		respChan:       respChan,
		unixSocketPath: unixSocketPath,
		resetConn:      &resetConn,
	}, nil
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

func (networkLayer *NetworkLayer) RegisterLogConsumer(consumer func(string)) string {
	uid := uuid.New().String()
	networkLayer.logCallbacks.Store(uid, consumer)
	return uid
}

func (networkLayer *NetworkLayer) UnregisterLogConsumer(uid string) {
	networkLayer.logCallbacks.Delete(uid)
}

func (networkLayer *NetworkLayer) GetRespChan() chan<- *protocol.Message {
	return networkLayer.respChan
}

func (networkLayer *NetworkLayer) RunAsync(ctx context.Context) {
	go func() {
		for {
			if networkLayer.resetConn.Load() {
				networkLayer.ResetConn()
				networkLayer.resetConn.Store(false)
			}

			decoder := json.NewDecoder(networkLayer.UnixConn)
			protoMsg := protocol.Message{}
			err := decoder.Decode(&protoMsg)
			if err != nil {
				//logger.ErrorErr(err, "Failed to read message from unix socket")
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}

			networkLayer.logCallbacks.Range(func(key, value any) bool {
				f := value.(func(string))
				f(protoMsg.LogMessage)
				return true
			})

			select {
			case <-ctx.Done():
				return
			case networkLayer.handleChan <- Message{
				Message: &protoMsg,
				resetConnFunc: func() {
					networkLayer.resetConn.Store(true)
				},
			}:
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case response := <-networkLayer.respChan:
				encoder := json.NewEncoder(networkLayer.UnixConn)
				logger.Debug("Responding with response '%s'", response.String())
				err := encoder.Encode(&response)
				if err != nil {
					logger.ErrorErr(err, "Failed to marshal DA response to bytes")
					return
				}

				logger.Debug("Sent DA response")
			}
		}
	}()
}

func (networkLayer *NetworkLayer) Close() error {
	return networkLayer.UnixConn.Close()
}
