package main

import (
	"html/template"

	"github.com/gin-gonic/contrib/gzip"
	"github.com/gin-gonic/contrib/jwt"
	"github.com/gin-gonic/gin"
)

var jWTokenKey = `aylbxm"XA1A*32:d.rvgNSS_RK3r;Tp`    //TODO: lets make a new key and put this somewhere safer!
var emailTokenKey = `yrCuch/BE*RN??tGUR?{CTYUTs_ApLW` //TODO: lets make a new key and put this somewhere safer!

func init() {

}

func main() {

	db := setupDB()
	router := gin.Default()
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	html := template.New("")
	html.Delims("{%", "%}")
	html.ParseFiles("index.html")

	router.SetHTMLTemplate(html)

	//Login
	users := router.Group("/users")
	users.Use(mapDB(&db))
	{
		users.GET("/:ID", func(c *gin.Context) {
			c.HTML(200, "index.html", nil)
		})
		users.POST("/login", login)
		users.POST("/signup", signup)
		users.POST("/verify", verify)
	}

	//API
	api := router.Group("/api/v1")
	api.Use(mapDB(&db))
	api.Use(jwt.Auth(jWTokenKey, "HS256"))
	{
		//create new
		api.POST("/users/:ID/streams", createStream)
		api.POST("/users/:ID/triggers", createTrigger)
		api.POST("/users/:ID/streams/:STREAMID/data", createDataPoint)

		//get data
		api.GET("/users/:ID", getUser)
		api.GET("/streams/:STREAMID", getStream) //shortcut for path of below
		api.GET("/users/:ID/streams/:STREAMID", getStream)

		//api.GET("/users/:ID/streams/:STREAMID", getStream)

		//get All
		api.GET("/users/:ID/streams", getAllStreams)
		api.GET("/users/:ID/triggers", getAllTriggers)
		api.GET("/users/:ID/streams/:STREAMID/data", getAllDataPoints)

		//modify existing
		//api.PUT("/users/:ID/triggers/:TRIGGERID", modifyTrigger)
		api.PUT("/users/:ID/streams/:STREAMID", addStream)

		//delete
		api.DELETE("/users/:ID", deleteUser)
		//api.DELETE("/users/:ID/triggers/:TRIGGERID", deleteTrigger)
		api.DELETE("/users/:ID/streams/:STREAMID", deleteStream)

		//websocket
		api.GET("/users/:ID/streams/:STREAMID/socket", handleWebSocket)
	}

	router.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	router.Static("/assets", "./assets")

	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	router.Run(":8000") // listen and serve on 0.0.0.0:8080
}
