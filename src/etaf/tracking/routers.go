/*
 * Netaf_Tracking
 *
 * ETAF Tracking Service
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package tracking

import (
	"free5gc/lib/logger_util"
	"free5gc/src/etaf/logger"
	"net/http"
	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

var HttpLog *logrus.Entry

func init() {
	HttpLog = logger.HttpLog
}

// Route is the information for every URI.
type Route struct {
	// Name is the name of this Route.
	Name string
	// Method is the string for the HTTP method. ex) GET, POST etc..
	Method string
	// Pattern is the pattern of the URI.
	Pattern string
	// HandlerFunc is the handler function of this route.
	HandlerFunc gin.HandlerFunc
}

// Routes is the list of the generated Route.
type Routes []Route

// NewRouter returns a new router.
func NewRouter() *gin.Engine {
	router := logger_util.NewGinWithLogrus(logger.GinLog)
	AddService(router)
	return router
}

func AddService(engine *gin.Engine) *gin.RouterGroup {
	group := engine.Group("/netaf-track/v1")

	for _, route := range routes {
		switch route.Method {
		case "GET":
			group.GET(route.Pattern, route.HandlerFunc)
		case "POST":
			group.POST(route.Pattern, route.HandlerFunc)
		case "PUT":
			group.PUT(route.Pattern, route.HandlerFunc)
		case "DELETE":
			group.DELETE(route.Pattern, route.HandlerFunc)
		}
	}
	return group
}

// Index is the index handler.
func Index(c *gin.Context) {
	c.String(http.StatusOK, "Emergency Tracking & Alert Function!")
}

var routes = Routes{
	{
		"Index",
		"GET",
		"/",
		Index,
	},

	// {
	// 	"N1N2MessageTransfer",
	// 	strings.ToUpper("Post"),
	// 	"/ue-contexts/:ueContextId/n1-n2-messages",
	// 	HTTPN1N2MessageTransfer,
	// },

}
