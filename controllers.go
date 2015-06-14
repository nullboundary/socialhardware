package main

import (
	"bitbucket.org/cicadaDev/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

func ping(c *gin.Context) {
	c.String(200, "pong")
}

func createNetwork(c *gin.Context) {
	db := getDB(c)

	newNet := newNetwork()

	c.BindJSON(&newNet)
	newNet.NetworkID = fmt.Sprintf("%d", utils.GenerateFnvHashID(time.Now().String()))

	err := db.Add("networks", newNet)
	if err != nil {
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}
	//all ok
	//email authtoken
	//newNet.AuthToken := utils.GenerateToken(tokenPrivateKey, serialNum, dlPass.KeyDoc.PassTypeIdentifier)

	c.JSON(http.StatusCreated, gin.H{"message": newNet.NetworkID, "status": http.StatusCreated})
}

func getNetwork(c *gin.Context) {
	db := getDB(c)
	netID := c.Param("ID")
	net := newNetwork()

	if ok, _ := db.FindById("networks", netID, &net); !ok {
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//render network struct
	c.JSON(http.StatusOK, net)
}

func deleteNetwork(c *gin.Context) {
	db := getDB(c)
	netID := c.Param("ID")

	err := db.DelById("networks", netID)
	if err != nil {
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//render stream struct
	c.JSON(http.StatusNoContent, gin.H{"message": http.StatusText(http.StatusNoContent), "status": http.StatusNoContent})
}

func createStream(c *gin.Context) {
	db := getDB(c)
	newStream := newStream()
	newStream.StreamAccess = true //default to public, unless reset in bind below
	c.BindJSON(&newStream)
	netID := c.Param("ID")
	newStream.NetworkID = netID
	newStream.StreamID = fmt.Sprintf("%d", utils.GenerateFnvHashID(time.Now().String()))

	err := db.Add("streams", newStream)
	if err != nil {
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	_, err = db.ArrayAppend("networks", netID, "networkStreams", newStream.StreamID)
	if err != nil {
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//all ok
	c.JSON(http.StatusCreated, gin.H{"message": newStream.StreamID, "status": http.StatusCreated})
}

//addStream adds a pre-existing public stream to a network
func addStream(c *gin.Context) {
	db := getDB(c)
	netID := c.Param("ID")
	streamID := c.Param("STREAMID")
	stream := newStream()

	if ok, _ := db.FindById("streams", streamID, &stream); !ok {
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	_, err := db.ArrayAppend("networks", netID, "networkStreams", streamID)
	if err != nil {
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//all ok
	c.JSON(http.StatusOK, gin.H{"message": streamID, "status": http.StatusOK})
}

func getStream(c *gin.Context) {
	db := getDB(c)
	streamID := c.Param("STREAMID")
	stream := newStream()

	if ok, _ := db.FindById("streams", streamID, &stream); !ok {
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//render stream struct
	c.JSON(http.StatusOK, stream)
}

func getAllStreams(c *gin.Context) {
	db := getDB(c)
	netID := c.Param("ID")
	streamList := []stream{}

	filter := map[string]string{"field": "networkId", "value": netID}

	//found false continues with empty struct. Error returns error message.
	_, err := db.FindAllEq("streams", filter, &streamList)
	if err != nil {
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//render streams list
	c.JSON(http.StatusOK, streamList)
}

func deleteStream(c *gin.Context) {
	db := getDB(c)
	streamID := c.Param("STREAMID")

	err := db.DelById("streams", streamID)
	if err != nil {
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//render stream struct
	c.JSON(http.StatusNoContent, gin.H{"message": http.StatusText(http.StatusNoContent), "status": http.StatusNoContent})
}

func createDataPoint(c *gin.Context) {

	streamID := c.Param("STREAMID")
	//networkID := c.Param("ID")
	/*
		if !accessToken(c.Request.Header.Get("Authorization"), networkID, streamID) {
			log.Printf("[WARN] access token unauthorized: %s", c.Request.Header.Get("Authorization"))
			status := http.StatusUnauthorized
			c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
			return
		} */

	db := getDB(c)

	dataPoint := newDataPoint()
	c.BindJSON(&dataPoint)

	dataPoint.StreamID = streamID
	//always need a timestamp
	nullTime := time.Time{}
	if dataPoint.TimeStamp == nullTime {
		dataPoint.TimeStamp = time.Now()
	}

	err := db.Add("dataPoints", dataPoint)
	if err != nil {
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}
	//all ok
	c.JSON(http.StatusCreated, gin.H{"message": dataPoint.TimeStamp.String(), "status": http.StatusCreated})
}

func getAllDataPoints(c *gin.Context) {
	db := getDB(c)
	streamID := c.Param("STREAMID")
	dataPoints := []dataPoint{}
	filter := map[string]string{"field": "streamId", "value": streamID}

	//found false continues with empty struct. Error returns error message.
	_, err := db.FindAllEq("dataPoints", filter, &dataPoints)
	if err != nil {
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//render datapoint list
	c.JSON(http.StatusOK, dataPoints)
}

/*
type connection struct {
	socket  *websocket.Conn  // The websocket connection.
	send    chan interface{} // Buffered channel of outbound messages.
	wsClose chan bool
}*/

func handleWebSocket(c *gin.Context) {

	db := getDB(c)
	streamID := c.Param("STREAMID")

	//upgrade to websocket connection
	ws, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println("Failed to set websocket upgrade: %+v", err)
		return
	}
	defer ws.Close()

	go reader(ws)

	//for {

	feedData, err := db.ChangesByIdx("dataPoints", "streamId", streamID, 1) //TODO: Filter per stream or network
	if err != nil {
		fmt.Printf("[ERROR] feedData: %s", err)
	}
	defer feedData.Close()

	var data dataPoint

	for feedData.Next(&data) { //loops forever
		ws.SetWriteDeadline(time.Now().Add(time.Second * 10))
		err := ws.WriteJSON(data)
		if err != nil {
			fmt.Println("[ERROR] writeJSON: %s", err.Error())
			return
		}
	}
	if feedData.Err() != nil {
		fmt.Println(feedData.Err())
	}
	//}
	//make a struct with buffered channel and websocket conn
	//conn := &connection{send: make(chan interface{}, 256), wsClose: make(chan bool, 1), socket: ws}
	//conn.wsClose <- false

	//db.ChangesChan(conn.send, "dataPoints", 1)
	/*go func() {
		for {
			//keep watching for changes
			feedData, err := db.Changes("dataPoints", 1)
			if err != nil {
				fmt.Printf("[ERROR] %s", err)
			}
			var data dataPoint
			for feedData.Next(&data) { //loops forever
				conn.send <- data //add data to send channel

				if <-conn.wsClose == true {
					fmt.Println("Exit changes routine")
					feedData.Close()
					return
				}
			}
			if feedData.Err() != nil {
				fmt.Println(feedData.Err())
			}

		}
	}()*/

	//go conn.writer()
	//conn.reader()
}

func reader(ws *websocket.Conn) {
	ws.SetReadDeadline(time.Now().Add(time.Second * 60))
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			fmt.Println("[ERROR] readMsg: %s", err.Error())
			break
		}
	}
}

/*
func (c *connection) writer() {
	for change := range c.send {
		changeData := change.(dataPoint)
		err := c.socket.WriteJSON(changeData)
		if err != nil {
			fmt.Println(err.Error())
			break
		}
	}
	c.socket.Close()
}*/
/*
func (c *connection) reader() {
	fmt.Println("reader")
	for {
		_, _, err := c.socket.ReadMessage()
		if err != nil {
			break
		}
	}
	c.socket.Close() */
/*for {
	if _, _, err := c.socket.NextReader(); err != nil {
		fmt.Println("nextReader: socket close")
		c.wsClose <- true
		c.socket.Close()
		break
	}
}*/
//}

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
