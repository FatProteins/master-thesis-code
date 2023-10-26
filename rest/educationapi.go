package rest

import (
	"context"
	"fmt"
	"github.com/FatProteins/master-thesis-code/consensus"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/FatProteins/master-thesis-code/network"
	"github.com/FatProteins/master-thesis-code/network/protocol"
	"github.com/FatProteins/master-thesis-code/setup"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var educationLogger = daLogger.NewLogger("educationapi")
var upgrader = websocket.Upgrader{
	ReadBufferSize:  65536,
	WriteBufferSize: 65536,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type nodeState int

var clientLogConsumer = sync.Map{}
var currentState = atomic.Value{}
var stepByStepModeEnabled = atomic.Bool{}
var stateUpdateChan = make(chan struct{}, 1)

const (
	online nodeState = iota
	paused
	stopped
)

func init() {
	currentState.Store(online)
}

func getCurrentState() (string, bool) {
	state := currentState.Load().(nodeState)
	stepByStep := stepByStepModeEnabled.Load()
	if state == online {
		return "online", stepByStep
	}
	if state == paused {
		return "paused", stepByStep
	}

	return "stopped", stepByStep
}

func addToClientLog(log string) {
	clientLogConsumer.Range(func(key, value any) bool {
		consumer := value.(func(string))
		consumer(log)
		return true
	})
}

func addClientLogConsumer(consumer func(string)) string {
	uid := uuid.New().String()
	clientLogConsumer.Store(uid, consumer)
	return uid
}

func removeClientLogConsumer(uid string) {
	clientLogConsumer.Delete(uid)
}

func notifyState(c *gin.Context) {
	select {
	case stateUpdateChan <- struct{}{}:
	default:
	}
}

func EducationApi(networkLayer *network.NetworkLayer, actionPicker *setup.ActionPicker, client consensus.ConsensusClient) {
	router := gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowWebSockets = true
	corsConfig.AllowAllOrigins = true
	router.Use(cors.New(corsConfig))
	router.POST("/education/action", func(c *gin.Context) { executeAction(c, networkLayer, actionPicker) }, notifyState)
	router.Any("/education/get-kv", func(c *gin.Context) { kvSubscription(c, client) })
	router.POST("/education/put-kv", func(c *gin.Context) { storeKeyValue(c, client) }, notifyState)
	router.POST("/education/delete-kv", func(c *gin.Context) { deleteKeyValue(c, client) }, notifyState)
	router.Any("/education/subscribe-log", func(c *gin.Context) { logSubscription(c, networkLayer) })
	router.POST("/education/step-by-step/toggle", func(c *gin.Context) { toggleStepByStep(c, networkLayer, actionPicker) }, notifyState)
	router.POST("/education/step-by-step/next-step", func(c *gin.Context) { nextStep(c, networkLayer, actionPicker) }, notifyState)
	router.Any("/education/subscribe-client-log", func(c *gin.Context) { clientLogSubscription(c) })
	router.Any("/education/subscribe-state", func(c *gin.Context) { subscribeToState(c, networkLayer, client) })
	router.GET("/education/get-state", func(c *gin.Context) { getState(c) })
	err := router.Run(":8080")
	if err != nil {
		panic(err)
	}
}

func getState(c *gin.Context) {
	state, isStepByStep := getCurrentState()
	c.JSON(200, StatusResponse{
		CurrentState:   state,
		StepByStepMode: isStepByStep,
	})
}

func subscribeToState(c *gin.Context, networkLayer *network.NetworkLayer, client consensus.ConsensusClient) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		educationLogger.ErrorErr(err, "Failed to establish Log websocket connection")
		return
	}
	defer conn.Close()

	for {
		status, err := client.GetStatus()
		if err != nil {
			err = conn.WriteJSON(NodeStatusUpdate{StatusError: true})
			if err != nil {
				educationLogger.ErrorErr(err, "Could not write to NodeStatus websocket")
				return
			}
		} else {
			err = conn.WriteJSON(NodeStatusUpdate{
				MemberState:  networkLayer.GetNodeState().String(),
				NodeId:       status.NodeId,
				Leader:       status.Leader,
				Term:         status.Term,
				Index:        status.Index,
				AppliedIndex: status.AppliedIndex,
				StatusError:  false,
			})
			if err != nil {
				educationLogger.ErrorErr(err, "Could not write to NodeStatus websocket")
				return
			}
		}

		select {
		case <-time.After(time.Second):
		case <-stateUpdateChan:
		}
	}
}

func executeAction(c *gin.Context, networkLayer *network.NetworkLayer, actionPicker *setup.ActionPicker) {
	var actionType ActionTypeRequest
	if err := c.BindJSON(&actionType); err != nil {
		educationLogger.ErrorErr(err, "Could not read ActionTypeRequest body")
		c.AbortWithError(500, err)
		return
	}

	if at, exists := protocol.ActionType_value[actionType.ActionType]; exists {
		protocolAt := protocol.ActionType(at)
		action := actionPicker.GetAction(protocolAt)
		err := action.Perform(func() { networkLayer.SetResetConn(true) }, networkLayer.GetRespChan())
		if err != nil {
			educationLogger.ErrorErr(err, "Failed to execute ection '5s'", actionType.ActionType)
			c.AbortWithError(500, err)
			return
		}

		if protocolAt == protocol.ActionType_PAUSE_ACTION_TYPE {
			currentState.Store(paused)
		} else if protocolAt == protocol.ActionType_STOP_ACTION_TYPE {
			currentState.Store(stopped)
		} else {
			currentState.Store(online)
		}

		c.Status(200)
	} else {
		educationLogger.Error("Could not find Action with name '%s'", actionType.ActionType)
		c.AbortWithStatus(500)
	}
}

func kvSubscription(c *gin.Context, client consensus.ConsensusClient) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		educationLogger.ErrorErr(err, "Failed to establish KV Store websocket connection")
		return
	}

	ch, err := client.SubscribeToChanges(context.TODO())
	if err != nil {
		educationLogger.ErrorErr(err, "Could not subscribe to KV changes")
		return
	}

	defer conn.Close()
	var changeLog []string
	for {
		var kvPairs []KVPair

		err = client.GetKV(func(key string, value string) {
			kvPairs = append(kvPairs, KVPair{Key: key, Value: value})
		})
		if err != nil {
			educationLogger.ErrorErr(err, "Could not get all KV")
		} else {
			err := conn.WriteJSON(KVUpdateResponse{Pairs: kvPairs, ChangeLog: changeLog})
			if err != nil {
				educationLogger.ErrorErr(err, "Could not write KV response")
			}
		}

		changes, ok := <-ch
		if !ok {
			educationLogger.Info("Cancelled KV subscription")
			return
		}

		changeLog = changes
	}
}

func logSubscription(c *gin.Context, networkLayer *network.NetworkLayer) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		educationLogger.ErrorErr(err, "Failed to establish Log websocket connection")
		return
	}
	defer conn.Close()

	finChan := make(chan struct{})
	consumerId := networkLayer.RegisterLogConsumer(func(logMsg string) {
		err := conn.WriteMessage(websocket.TextMessage, []byte(logMsg))
		if err != nil {
			educationLogger.ErrorErr(err, "Could not write to Log websocket")
			close(finChan)
			return
		}
	})
	defer networkLayer.UnregisterLogConsumer(consumerId)

	<-finChan
}

func clientLogSubscription(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		educationLogger.ErrorErr(err, "Failed to establish Client Log websocket connection")
		return
	}

	finChan := make(chan struct{})
	consumerId := addClientLogConsumer(func(logMsg string) {
		err := conn.WriteMessage(websocket.TextMessage, []byte(logMsg))
		if err != nil {
			educationLogger.ErrorErr(err, "Could not write to Client Log websocket")
			close(finChan)
			return
		}
	})
	defer removeClientLogConsumer(consumerId)

	<-finChan
}

func storeKeyValue(c *gin.Context, client consensus.ConsensusClient) {
	var kvPair KVPair
	if err := c.BindJSON(&kvPair); err != nil {
		educationLogger.ErrorErr(err, "Could not read StoreKV body")
		c.AbortWithError(500, err)
		return
	}

	err := client.StoreKV(kvPair.Key, kvPair.Value)
	if err != nil {
		educationLogger.ErrorErr(err, "Could not put KV")
		if strings.Contains(err.Error(), "context deadline exceeded") {
			addToClientLog(fmt.Sprintf("PUT '%s' '%s' TIMEOUT.", kvPair.Key, kvPair.Value))
		} else {
			addToClientLog(fmt.Sprintf("PUT '%s' '%s' FAILURE: %s.", kvPair.Key, kvPair.Value, err.Error()))
		}

		c.AbortWithError(500, err)
		return
	}

	addToClientLog(fmt.Sprintf("PUT '%s' '%s' SUCCESS.", kvPair.Key, kvPair.Value))
	c.Status(200)
}

func deleteKeyValue(c *gin.Context, client consensus.ConsensusClient) {
	var keyDelete KeyDelete
	if err := c.BindJSON(&keyDelete); err != nil {
		educationLogger.ErrorErr(err, "Could not read KeyDelete body")
		if strings.Contains(err.Error(), "context deadline exceeded") {
			addToClientLog(fmt.Sprintf("DELETE '%s' TIMEOUT.", keyDelete.Key))
		} else {
			addToClientLog(fmt.Sprintf("DELETE '%s' FAILURE: %s.", keyDelete.Key, err.Error()))
		}
		c.AbortWithError(500, err)
		return
	}

	err := client.DeleteKey(keyDelete.Key)
	if err != nil {
		educationLogger.ErrorErr(err, "Could not delete key")
		c.AbortWithError(500, err)
		return
	}

	addToClientLog(fmt.Sprintf("DELETE '%s' SUCCESS.", keyDelete.Key))
	c.Status(200)
}

func toggleStepByStep(c *gin.Context, networkLayer *network.NetworkLayer, actionPicker *setup.ActionPicker) {
	var stepByStepRequest StepByStepRequest
	if err := c.BindJSON(&stepByStepRequest); err != nil {
		educationLogger.ErrorErr(err, "Could not read StepByStepRequest body")
		c.AbortWithError(500, err)
		return
	}

	actionPicker.SetStepByStepMode(stepByStepRequest.Enable)
	if !stepByStepRequest.Enable {
		action := actionPicker.GetAction(protocol.ActionType_CONTINUE_ACTION_TYPE)
		action.Perform(func() { networkLayer.SetResetConn(true) }, networkLayer.GetRespChan())
	}

	stepByStepModeEnabled.Store(stepByStepRequest.Enable)
	c.Status(200)
}

func nextStep(c *gin.Context, networkLayer *network.NetworkLayer, actionPicker *setup.ActionPicker) {
	action := actionPicker.GetAction(protocol.ActionType_CONTINUE_ACTION_TYPE)
	action.Perform(func() { networkLayer.SetResetConn(true) }, networkLayer.GetRespChan())
	c.Status(200)
}
