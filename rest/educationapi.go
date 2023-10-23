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
	router.POST("/education/action", func(ctx *gin.Context) { executeAction(ctx, networkLayer, actionPicker) })
	router.Any("/education/get-kv", func(ctx *gin.Context) { kvSubscription(ctx, client) })
	router.POST("/education/put-kv", func(ctx *gin.Context) { storeKeyValue(ctx, client) })
	router.POST("/education/delete-kv", func(ctx *gin.Context) { deleteKeyValue(ctx, client) })
	err := router.Run(":8080")
	if err != nil {
		panic(err)
	}
}

func executeAction(context *gin.Context, networkLayer *network.NetworkLayer, actionPicker *setup.ActionPicker) {
	var actionType ActionTypeRequest
	if err := context.BindJSON(&actionType); err != nil {
		educationLogger.ErrorErr(err, "Could not read ActionTypeRequest body")
		context.AbortWithError(500, err)
		return
	}

	if at, exists := protocol.ActionType_value[actionType.ActionType]; exists {
		action := actionPicker.GetAction(protocol.ActionType(at))
		action.Perform(func() { networkLayer.SetResetConn(true) })
		context.Status(200)
	} else {
		educationLogger.Error("Could not find Action with name '%s'", actionType.ActionType)
		context.Status(500)
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

func storeKeyValue(context *gin.Context, client consensus.ConsensusClient) {
	var kvPair KVPair
	if err := context.BindJSON(&kvPair); err != nil {
		educationLogger.ErrorErr(err, "Could not read StoreKV body")
		return
	}

	err := client.StoreKV(kvPair.Key, kvPair.Value)
	if err != nil {
		educationLogger.ErrorErr(err, "Could not put KV")
		context.AbortWithError(500, err)
		return
	}

	context.Status(200)
}

func deleteKeyValue(context *gin.Context, client consensus.ConsensusClient) {
	var keyDelete KeyDelete
	if err := context.BindJSON(&keyDelete); err != nil {
		educationLogger.ErrorErr(err, "Could not read KeyDelete body")
		return
	}

	err := client.DeleteKey(keyDelete.Key)
	if err != nil {
		educationLogger.ErrorErr(err, "Could not delete key")
		context.AbortWithError(500, err)
		return
	}

	context.Status(200)
}
