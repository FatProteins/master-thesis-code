package network

import (
	"bufio"
	"context"
	"encoding/json"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/FatProteins/master-thesis-code/network/protocol"
	"github.com/FatProteins/master-thesis-code/util"
	"github.com/google/uuid"
	"net"
	"sync"
	"sync/atomic"
)

var logger = daLogger.NewLogger("network")

type NetworkLayer struct {
	*net.TCPConn
	localAddr      *net.TCPAddr
	messagePool    *util.Pool[protocol.Message]
	handleChan     chan<- Message
	respChan       chan *protocol.Message
	unixSocketPath string
	resetConn      *atomic.Bool
	logCallbacks   sync.Map
	nodeState      atomic.Value
}

func NewNetworkLayer(handleChan chan<- Message, respChan chan *protocol.Message, localAddr *net.TCPAddr, unixSocketPath string) (*NetworkLayer, error) {

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
	if networkLayer.TCPConn != nil {
		networkLayer.TCPConn.Close()
	}

	listener, err := net.ListenTCP("tcp", networkLayer.localAddr)
	if err != nil {
		panic("failed to listen to unix socket")
	}

	connection, err := listener.AcceptTCP()
	if err != nil {
		panic("failed to connect unix socket")
	}
	networkLayer.TCPConn = connection
}

func (networkLayer *NetworkLayer) GetNodeState() protocol.NodeState {
	nodeState := networkLayer.nodeState.Load()
	if nodeState == nil {
		return protocol.NodeState_FOLLOWER
	}

	return nodeState.(protocol.NodeState)
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
	var scanner *bufio.Scanner
	go func() {
		for {
			if networkLayer.resetConn.Load() {
				logger.Info("Resetting connection to Node")
				networkLayer.ResetConn()
				networkLayer.resetConn.Store(false)
				scanner = bufio.NewScanner(networkLayer.TCPConn)
			}

			scanner.Scan()
			err := scanner.Err()
			if err != nil {
				logger.ErrorErr(err, "Error reading from connection")
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}

			protoMsg := protocol.Message{}
			bytes := scanner.Bytes()
			err = json.Unmarshal(bytes, &protoMsg)
			if err != nil {
				//logger.ErrorErr(err, "Failed to read message from unix socket")
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}

			if protoMsg.MessageType == protocol.MessageType_STATE_LOG {
				networkLayer.nodeState.Store(protoMsg.NodeState)
				continue
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
				encoder := json.NewEncoder(networkLayer.TCPConn)
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
	return networkLayer.TCPConn.Close()
}
