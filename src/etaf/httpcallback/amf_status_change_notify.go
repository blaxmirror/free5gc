package httpcallback

import (
//	"free5gc/lib/http_wrapper"
//	"free5gc/lib/openapi"
//	"free5gc/lib/openapi/models"
	"free5gc/src/etaf/logger"
//	"free5gc/src/pcf/producer"
//	"net/http"
//
	"github.com/gin-gonic/gin"
)

func HTTPLocInfoNotify(c *gin.Context) {
	logger.CommLog.Info("Location Info Notification received")
}
