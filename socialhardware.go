package main

import (
	"github.com/gin-gonic/gin"
	"html/template"
)

func init() {

}

func main() {

	db := setupDB()
	router := gin.Default()

	//API
	api := router.Group("/api/v1")
	api.Use(mapDB(&db))
	{
		//create new
		api.POST("/networks", createNetwork)
		api.POST("/networks/:ID/streams", createStream)
		api.POST("/networks/:ID/streams/:STREAMID/data", createDataPoint)

		//get data
		api.GET("/networks/:ID", getNetwork)
		api.GET("/networks/:ID/streams", getAllStreams)
		api.GET("/networks/:ID/streams/:STREAMID", getStream)
		api.GET("/networks/:ID/streams/:STREAMID/data", getAllDataPoints)

		//modify existing
		api.PUT("/networks/:ID/streams/:STREAMID", addStream)

		//delete
		api.DELETE("/networks/:ID", deleteNetwork)
		api.DELETE("/networks/:ID/streams/:STREAMID", deleteStream)

		//websocket
		api.GET("/networks/:ID/streams/:STREAMID/socket", handleWebSocket)
	}

	router.Static("/assets", "./assets")

	//router.LoadHTMLFiles("index.html", "network.html")
	//router.StaticFile("/", "index.html")
	//router.StaticFile("/networks", "index.html")

	html := template.New("")
	html.Delims("{%", "%}")
	html.ParseFiles("index.html")

	router.SetHTMLTemplate(html)

	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})
	router.GET("/networks/:ID", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	router.GET("/ping", ping)
	router.Run(":8000") // listen and serve on 0.0.0.0:8080
}
