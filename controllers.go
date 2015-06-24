package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"bitbucket.org/cicadaDev/utils"
	"code.google.com/p/go.crypto/bcrypt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

//login is the handler for posting user login details
func login(c *gin.Context) {

	//check provided username/pass
	db := getDB(c)

	type loginForm struct {
		Email    string `json:"email" binding:"required"`    //the users email works as an id
		Password string `json:"password" binding:"required"` //users password string. not added to db!
	}

	loginInfo := &loginForm{} //store info coming from client form
	c.BindJSON(&loginInfo)

	user := newUser() //store user info retrieved from DB

	_, err := db.FindById("users", loginInfo.Email, user)
	if err != nil {
		log.Println(err)
		status := http.StatusUnauthorized
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	if err != nil || bcrypt.CompareHashAndPassword(user.PassCrypt, []byte(loginInfo.Password)) != nil {
		log.Println(err)
		status := http.StatusUnauthorized
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	jwt, err := createJWToken("token", []byte(jWTokenKey), loginInfo.Email)
	if err != nil {
		log.Println(err)
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//return token as json
	c.JSON(http.StatusOK, jwt)
}

//signup is the request handler for posting user signup details
func signup(c *gin.Context) {
	db := getDB(c)
	type loginForm struct {
		Email    string `json:"email" binding:"required"`    //the users email works as an id
		Password string `json:"password" binding:"required"` //users password string. not added to db!
	}

	loginInfo := &loginForm{} //store info coming from client form
	c.BindJSON(&loginInfo)

	user := newUser() //store user info retrieved from DB

	user.setPassword(loginInfo.Password)
	user.Email = loginInfo.Email
	user.Created = time.Now()
	user.Verified = false

	err := db.Add("users", user)
	if err != nil {
		log.Println(err)
		status := http.StatusConflict
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	expiration := strconv.FormatInt(user.Created.AddDate(0, 0, 1).Unix(), 10)        //token expires in 24 hours
	emailtoken := utils.GenerateToken([]byte(emailTokenKey), user.Email, expiration) //get token from base64 hmac
	url := createRawURL(emailtoken, user.Email, expiration, c.Request.Host)          //generate verification url
	emailVerify := NewEmailer()
	go emailVerify.Send(user.Email, emailtoken, url) //send concurrently

	c.JSON(http.StatusCreated, gin.H{"message": user.Email, "status": http.StatusCreated})
}

//verify is a handler the verifies email verification tokens.
func verify(c *gin.Context) {

	emailAddr := c.Query("email")
	emailToken := c.Query("token")
	emailExpire := c.Query("expires")

	_, err := utils.VerifyToken([]byte(emailTokenKey), emailToken, emailAddr, emailExpire)
	if err != nil {
		log.Println(err)
		status := http.StatusBadRequest
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": emailAddr, "status": http.StatusOK})
}

func createNetwork(c *gin.Context) {
	db := getDB(c)

	newNet := newNetwork()

	c.BindJSON(&newNet)
	newNet.NetworkID = fmt.Sprintf("%d", utils.GenerateFnvHashID(time.Now().String()))
	newNet.NetworkAdmin = 
	//newNet.NetworkStreams = []string{}
	//newNet.NetworkTriggers = []trigger{}

	err := db.Add("networks", newNet)
	if err != nil {
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": newNet.NetworkID, "status": http.StatusCreated})
}

//getNetwork returns a specific network as json
func getNetwork(c *gin.Context) {
	db := getDB(c)
	netID := c.Param("ID")
	net := newNetwork()

	if ok, _ := db.FindById("networks", netID, &net); !ok {
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	/*userID := c.MustGet("jwt").(string)

	if net.NetworkAdmin != userID {
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	} */

	//render network struct
	c.JSON(http.StatusOK, net)
}

//getAllNetworks returns all data networks for a particular user

/*
func getAllNetworks(c *gin.Context) {
	db := getDB(c)
	netID := c.Param("ID")
	networkList := []network{}

	filter := map[string]string{"field": "networkAdmin", "value": netID}

	//found false continues with empty struct. Error returns error message.
	_, err := db.FindAllEq("networks", filter, &networkList)
	if err != nil {
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//render streams list
	c.JSON(http.StatusOK, streamList)
} */

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

//createStream creates a new empty data stream and adds it to the current network
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
		log.Printf("[ERROR] %s", err)
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	_, err = db.ArrayAppend("networks", netID, "networkStreams", newStream.StreamID)
	if err != nil {
		log.Printf("[ERROR] %s", err)
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
		log.Printf("[ERROR] %s", err)
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//all ok
	c.JSON(http.StatusOK, gin.H{"message": streamID, "status": http.StatusOK})
}

//getStream returns a specific data stream as a json document
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

//getAllStreams returns all data streams for a particular network
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

//removeStream removes a specific data stream from current network
//The opposite of add stream.
func removeStream(c *gin.Context) {
	db := getDB(c)
	netID := c.Param("ID")
	streamID := c.Param("STREAMID")

	//TODO: Find index in array to delete by for trigger.
	_, err := db.ArrayDeleteAt("networks", netID, "networkStreams", 0)
	if err != nil {
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//return deleted message
	c.JSON(http.StatusNoContent, gin.H{"message": http.StatusText(http.StatusNoContent), "status": http.StatusNoContent})
}

//deleteStream deletes a specific data stream from the database
func deleteStream(c *gin.Context) {
	db := getDB(c)
	streamID := c.Param("STREAMID")

	err := db.DelById("streams", streamID)
	if err != nil {
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//TODO: delete stream from streamid array in network! Or leave it and display "No longer available"
	//TODO: Find index in array to delete by for trigger.
	_, err := db.ArrayDeleteAt("networks", netID, "networkStreams", 0)
	if err != nil {
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//return deleted message
	c.JSON(http.StatusNoContent, gin.H{"message": http.StatusText(http.StatusNoContent), "status": http.StatusNoContent})
}

//createDataPoint adds a new datapoint to the db for a particular stream
func createDataPoint(c *gin.Context) {

	streamID := c.Param("STREAMID")

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

	checkTriggers(db, dataPoint)

	//all ok
	c.JSON(http.StatusCreated, gin.H{"message": dataPoint.TimeStamp.String(), "status": http.StatusCreated})
}

//getAllDataPoints retrieves all datapoints for a particular data stream
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

//createTrigger adds a new conditional trigger to a network
func createTrigger(c *gin.Context) {
	db := getDB(c)
	newTrigger := newTrigger()

	c.BindJSON(&newTrigger)
	newTrigger.TriggerID = fmt.Sprintf("%d", utils.GenerateFnvHashID(time.Now().String()))

	netID := c.Param("ID")

	_, err := db.ArrayAppend("networks", netID, "networkTriggers", newTrigger)
	if err != nil {
		log.Printf("[ERROR] %s", err)
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//all ok
	c.JSON(http.StatusCreated, gin.H{"message": newTrigger.TriggerID, "status": http.StatusCreated})
}

//getAllTriggers returns a list of all triggers for a network
func getAllTriggers(c *gin.Context) {
	db := getDB(c)
	netID := c.Param("ID")
	net := newNetwork()

	if ok, _ := db.FindById("networks", netID, &net); !ok {
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	triggerList := net.NetworkTriggers //better to filter in db?

	//render trigger list
	c.JSON(http.StatusOK, triggerList)

}

//modifyTrigger changes a triggers values/settings
func modifyTrigger(c *gin.Context) {
	db := getDB(c)
	netID := c.Param("ID")

	triggerID := c.Param("TRIGGERID")
	trigger := newTrigger()
	c.BindJSON(&trigger)

	//TODO: find trigger in network trigger array. Replace it.
	_, err := db.ArrayAppend("networks", netID, "networkTriggers", trigger)
	if err != nil {
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//all ok
	c.JSON(http.StatusOK, gin.H{"message": triggerID, "status": http.StatusOK})
}

//deleteTrigger removes a pre-existing trigger from a network
func deleteTrigger(c *gin.Context) {
	db := getDB(c)
	netID := c.Param("ID")
	//triggerID := c.Param("TRIGGERID")
	//trigger := newStream()

	//TODO: Find index in array to delete by for trigger.
	_, err := db.ArrayDeleteAt("networks", netID, "networkTriggers", 0)
	if err != nil {
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//return deleted message
	c.JSON(http.StatusNoContent, gin.H{"message": http.StatusText(http.StatusNoContent), "status": http.StatusNoContent})
}

//handleWebSocket handles incoming requests to setup a websocket
//and return live data updates for a particular stream
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

}

//reader is used to read from a websocket
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

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
