package rest

import (
	"context"
	"github.com/FatProteins/master-thesis-code/consensus"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/FatProteins/master-thesis-code/network"
	"github.com/FatProteins/master-thesis-code/network/protocol"
	"github.com/FatProteins/master-thesis-code/setup"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
)

var educationLogger = daLogger.NewLogger("educationapi")
var upgrader = websocket.Upgrader{
	ReadBufferSize:  65536,
	WriteBufferSize: 65536,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func EducationApi(networkLayer *network.NetworkLayer, actionPicker *setup.ActionPicker, client consensus.ConsensusClient) {
	router := gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowWebSockets = true
	corsConfig.AllowAllOrigins = true
	router.Use(cors.New(corsConfig))
	router.POST("/education/action", func(c *gin.Context) { executeAction(c, networkLayer, actionPicker) })
	router.Any("/education/get-kv", func(c *gin.Context) { kvSubscription(c, client) })
	router.POST("/education/put-kv", func(c *gin.Context) { storeKeyValue(c, client) })
	router.POST("/education/delete-kv", func(c *gin.Context) { deleteKeyValue(c, client) })
	router.Any("/education/subscribe-log", func(c *gin.Context) { logSubscription(c, networkLayer) })
	router.POST("/education/step-by-step/toggle", func(c *gin.Context) { toggleStepByStep(c, networkLayer, actionPicker) })
	router.POST("/education/step-by-step/next-step", func(c *gin.Context) { nextStep(c, networkLayer, actionPicker) })
	err := router.Run(":8080")
	if err != nil {
		panic(err)
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
		action := actionPicker.GetAction(protocol.ActionType(at))
		action.Perform(func() { networkLayer.SetResetConn(true) }, networkLayer.GetRespChan())
		c.Status(200)
	} else {
		educationLogger.Error("Could not find Action with name '%s'", actionType.ActionType)
		c.Status(500)
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
	for {
		var kvPairs []KVPair

		err = client.GetKV(func(key string, value string) {
			kvPairs = append(kvPairs, KVPair{Key: key, Value: value})
		})
		if err != nil {
			educationLogger.ErrorErr(err, "Could not get all KV")
			return
		}

		err := conn.WriteJSON(AllKVResponse{Pairs: kvPairs})
		if err != nil {
			educationLogger.ErrorErr(err, "Could not write KV response")
			return
		}

		_, ok := <-ch
		if !ok {
			educationLogger.Info("Cancelled KV subscription")
			return
		}
	}
}

func logSubscription(c *gin.Context, networkLayer *network.NetworkLayer) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		educationLogger.ErrorErr(err, "Failed to establish KV Store websocket connection")
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
		c.AbortWithError(500, err)
		return
	}

	c.Status(200)
}

func deleteKeyValue(c *gin.Context, client consensus.ConsensusClient) {
	var keyDelete KeyDelete
	if err := c.BindJSON(&keyDelete); err != nil {
		educationLogger.ErrorErr(err, "Could not read KeyDelete body")
		c.AbortWithError(500, err)
		return
	}

	err := client.DeleteKey(keyDelete.Key)
	if err != nil {
		educationLogger.ErrorErr(err, "Could not delete key")
		c.AbortWithError(500, err)
		return
	}

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
		c.Status(200)
	}
	c.Status(200)
}

func nextStep(c *gin.Context, networkLayer *network.NetworkLayer, actionPicker *setup.ActionPicker) {
	action := actionPicker.GetAction(protocol.ActionType_CONTINUE_ACTION_TYPE)
	action.Perform(func() { networkLayer.SetResetConn(true) }, networkLayer.GetRespChan())
	c.Status(200)
}
