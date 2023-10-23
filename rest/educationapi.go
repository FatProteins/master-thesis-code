package rest

import (
	"github.com/FatProteins/master-thesis-code/consensus"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/FatProteins/master-thesis-code/network"
	"github.com/FatProteins/master-thesis-code/network/protocol"
	"github.com/FatProteins/master-thesis-code/setup"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var educationLogger = daLogger.NewLogger("educationapi")

func EducationApi(networkLayer *network.NetworkLayer, actionPicker *setup.ActionPicker, client consensus.ConsensusClient) {
	router := gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowWebSockets = true
	router.Use(cors.New(corsConfig))
	router.POST("/education/action", func(ctx *gin.Context) { executeAction(ctx, networkLayer, actionPicker) })
	router.POST("/education/get-kv", func(ctx *gin.Context) { getAllKV(ctx, client) })
	router.POST("/education/store-kv", func(ctx *gin.Context) { storeKeyValue(ctx, client) })
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

func getAllKV(context *gin.Context, client consensus.ConsensusClient) {
	var key Key
	if err := context.BindJSON(&key); err != nil {
		educationLogger.ErrorErr(err, "Could not read Key body")
		return
	}

	var kvPairs []KVPair
	err := client.GetAllKV(func(key string, value string) {
		kvPairs = append(kvPairs, KVPair{Key: key, Value: value})
	})
	if err != nil {
		educationLogger.ErrorErr(err, "Could not get all KV")
		context.AbortWithError(500, err)
		return
	}

	context.JSON(200, AllKVResponse{Pairs: kvPairs})
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
