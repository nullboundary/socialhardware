package main

import (
	"html/template"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/contrib/gzip"
	"github.com/gin-gonic/contrib/jwt"
	"github.com/gin-gonic/gin"
)

var (
	jWTokenKey    string
	emailTokenKey string
	rethinkKey    string
)

func init() {
	jWTokenKey = os.Getenv("SOCIALHW_JWTKEY")
	emailTokenKey = os.Getenv("SOCIALHW_EMAILKEY")
	rethinkKey = os.Getenv("SOCIALHW_DBKEY")
}

func main() {

	db := setupDB()
	m := newMqttClient()
	m.setup("socialhardware", "tcp://localhost:1883", 0, db)
	m.registerStreams(db)

	router := gin.Default()

	//Add Gzip middleware
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	//add logrus middleware
	logger := log.New()
	logger.Level = log.DebugLevel
	router.Use(ginrus.Ginrus(logger, time.RFC3339, false))

	html := template.New("")
	html.Delims("{%", "%}")
	html.ParseFiles("index.html")

	router.SetHTMLTemplate(html)

	//Login
	users := router.Group("/auth")
	users.Use(mapDB(db))
	{
		users.GET("/verify", verify)
		users.POST("/login", login)
		users.POST("/signup", signup)
	}

	//API
	api := router.Group("/api/v1")
	api.Use(mapDB(db))
	api.Use(mapMQTT(m))
	api.Use(jwt.Auth(jWTokenKey, "HS256"))
	{
		//create new
		api.POST("/streams", createStream)
		api.POST("/triggers", createTrigger)
		api.POST("/streams/:STREAMID/data", createDataPoint)

		//get data
		api.GET("/users", getUser)
		api.GET("/streams/:STREAMID", getStream)
		api.GET("/triggers/:TRIGGERID", getTrigger)

		//api.GET("/users/:ID/streams/:STREAMID", getStream)

		//get All
		api.GET("/streams", getAllStreams)
		api.GET("/triggers", getAllTriggers)
		api.GET("/streams/:STREAMID/data", getAllDataPoints)

		//modify existing
		api.PUT("/streams/:STREAMID", addStream)
		api.PUT("/triggers/:TRIGGERID", modTrigger)

		//delete
		api.DELETE("/users/:ID", deleteUser)
		api.DELETE("/streams/:STREAMID", deleteStream)
		api.DELETE("/triggers/:TRIGGERID", deleteTrigger)

		//websocket
		api.GET("/streams/:STREAMID/socket", handleWebSocket)
	}

	router.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	router.Static("/assets", "./assets")

	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})
	//TODO: this should be a different path then users/username
	router.GET("users/:ID", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	router.Run(os.Getenv("SOCIALHW_PORT"))
}
