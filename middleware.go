package main

import (
	"bitbucket.org/cicadaDev/storer"
	"fmt"
	"github.com/gin-gonic/gin"
)

func mapDB(s *storer.ReThink) gin.HandlerFunc {

	return func(c *gin.Context) {
		fmt.Println("Set Context DB")
		c.Set("db", s)
		c.Next()
	}

}

func getDB(c *gin.Context) *storer.ReThink {
	return c.MustGet("db").(*storer.ReThink)
}

func setupDB() *storer.ReThink {
	rt := storer.NewReThink()

	fmt.Println("setup db conn pool")

	rt.Url = "127.0.0.1"
	rt.Port = "28015"
	rt.DbName = "socialhw"
	rt.AuthKey = rethinkKey

	rt.Conn()
	return rt
}

func mapMQTT(m *mqttClient) gin.HandlerFunc {

	return func(c *gin.Context) {
		fmt.Println("Set Context MQTT")
		c.Set("mqtt", m)
		c.Next()
	}

}

func getMQTT(c *gin.Context) *mqttClient {
	return c.MustGet("mqtt").(*mqttClient)
}
