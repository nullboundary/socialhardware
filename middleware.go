package main

import (
	"bitbucket.org/cicadaDev/storer"
	"fmt"
	"github.com/gin-gonic/gin"
)

func mapDB(s *storer.ReThink) gin.HandlerFunc {

	return func(c *gin.Context) {
		fmt.Println("Set Context DB")

		/*_, ok := c.Get("db")
		fmt.Println(ok)
		if !ok { //test is the db is already added
			//connect to db
			fmt.Println("Set Context DB")
			rt := storer.NewReThink()

			rt.Url = "127.0.0.1"
			rt.Port = "28015"
			rt.DbName = "socialhardware"

			s := *rt //storer.Storer(rt) //abstract cb to a Storer
			s.Conn()*/

		c.Set("db", s)
		c.Next()
		//}
	}

}

func getDB(c *gin.Context) *storer.ReThink {
	return c.MustGet("db").(*storer.ReThink)
}

func setupDB() storer.ReThink {
	rt := storer.NewReThink()

	fmt.Println("setup db conn pool")

	rt.Url = "127.0.0.1"
	rt.Port = "28015"
	rt.DbName = "socialhardware"

	s := *rt //storer.Storer(rt) //abstract cb to a Storer
	s.Conn()

	return s
}
