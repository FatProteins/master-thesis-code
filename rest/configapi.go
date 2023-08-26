package rest

import (
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/FatProteins/master-thesis-code/setup"
	"github.com/gin-gonic/gin"
)

var logger = daLogger.NewLogger("configapi")

func ConfigApi() {
	router := gin.Default()
	router.POST("/config/update", updateConfig)
}

func updateConfig(context *gin.Context) {
	var config setup.FaultConfig
	if err := context.BindYAML(&config); err != nil {
		logger.ErrorErr(err, "Could not read config update entity")
		return
	}

}
