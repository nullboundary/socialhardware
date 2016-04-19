package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"code.google.com/p/go.crypto/bcrypt"
	log "github.com/Sirupsen/logrus"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/nullboundary/utilbelt"
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
	logUser := log.WithField("user", loginInfo.Email)

	user := newUser() //store user info retrieved from DB

	_, err := db.FindByIdx("users", "email", loginInfo.Email, user)
	if err != nil {
		logUser.Errorf("login find user error %s", err)
		status := http.StatusUnauthorized
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	if user.Verified == false {
		logUser.Errorln("Email not verified")
		status := http.StatusUnauthorized
		c.JSON(status, gin.H{"message": "Please verify your email before login", "status": status})
		return
	}

	if err != nil || bcrypt.CompareHashAndPassword(user.PassCrypt, []byte(loginInfo.Password)) != nil {
		logUser.Errorf("Bcrypt hash compare error %s", err)
		status := http.StatusUnauthorized
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	jwt, err := createJWToken("token", []byte(jWTokenKey), user.ID)
	if err != nil {
		logUser.Errorf("create JWT error %s", err)
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//return token as json
	c.JSON(http.StatusOK, gin.H{"user": user.ID, "jwt": jwt})
}

//signup is the request handler for posting user signup details
func signup(c *gin.Context) {
	db := getDB(c)
	type loginForm struct {
		Email     string `json:"email" binding:"required"`    //the users email works as an id
		Password  string `json:"password" binding:"required"` //users password string. not added to db!
		InviteKey string `json:"invite" binding:"required"`   //users invite string.
	}

	loginInfo := &loginForm{} //store info coming from client form
	c.BindJSON(&loginInfo)
	logUser := log.WithField("user", loginInfo.Email)

	//verify invite token. Return if invite fails
	ok, err := utils.VerifyToken([]byte(emailTokenKey), loginInfo.InviteKey, loginInfo.Email)
	if !ok {
		if err != nil {
			log.WithFields(log.Fields{"email": loginInfo.Email}).Errorf("verify invite token error %s", err)
		}
		log.WithFields(log.Fields{"email": loginInfo.Email}).Errorln("failed verify invite token")
		status := http.StatusBadRequest
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	user := newUser() //store user info retrieved from DB

	user.ID = fmt.Sprintf("%d", utils.GenerateFnvHashID(time.Now().String()))
	user.setPassword(loginInfo.Password)
	user.Email = loginInfo.Email
	user.Created = time.Now()
	user.Verified = false

	//TODO: add one default stream for a new user

	err = db.Add("users", user)
	if err != nil {
		logUser.Errorf("db add user error %s", err)
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

	db := getDB(c)

	log.Debugln("verify user")

	emailAddr := c.Query("email")
	emailToken := c.Query("token")
	emailExpire := c.Query("expires")

	_, err := utils.VerifyToken([]byte(emailTokenKey), emailToken, emailAddr, emailExpire)
	if err != nil {
		log.WithFields(log.Fields{"email": emailAddr, "expire": emailExpire}).Errorf("verify email token error %s", err)
		status := http.StatusBadRequest
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	userData := newUser() //store user info retrieved from DB

	//TODO optimize with 1 db call

	_, err = db.FindByIdx("users", "email", emailAddr, userData)
	if err != nil {
		log.WithField("email", emailAddr).Errorf("verify find user error %s", err)
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	userData.Verified = true

	_, err = db.Merge("users", userData.ID, false, userData)
	if err != nil {
		log.WithField("userid", userData.ID).Errorf("verify merge user error %s", err)
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": emailAddr, "status": http.StatusOK})
}

//getUser returns a specific user as json
func getUser(c *gin.Context) {
	db := getDB(c)
	log.Debugln("get user")

	//userID := c.Param("ID")
	userData := newUser()

	jwToken := c.MustGet("jwt").(*jwt.Token)
	jwtUser := jwToken.Claims["sub"].(string)

	_, err := db.FindById("users", jwtUser, &userData)
	if err != nil {
		log.WithField("userid", jwtUser).Errorf("find user error %s", err)
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//render user struct
	c.JSON(http.StatusOK, userData)
}

//deleteUser
func deleteUser(c *gin.Context) {
	db := getDB(c)
	log.Debugln("delete user")
	//userID := c.Param("ID")

	jwToken := c.MustGet("jwt").(*jwt.Token)
	jwtUser := jwToken.Claims["sub"].(string)

	err := db.DelById("users", jwtUser)
	if err != nil {
		log.WithField("userid", jwtUser).Errorf("delete user error %s", err)
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//render stream struct
	c.JSON(http.StatusNoContent, gin.H{"message": http.StatusText(http.StatusNoContent), "status": http.StatusNoContent})
}

//createStream creates a new empty data stream and adds it to the current user account
func createStream(c *gin.Context) {
	db := getDB(c)
	m := getMQTT(c)

	//userID := c.Param("ID")
	jwToken := c.MustGet("jwt").(*jwt.Token)
	jwtUser := jwToken.Claims["sub"].(string)

	newStream := newStream()
	newStream.StreamAccess = true //default to public, unless reset in bind below

	c.BindJSON(&newStream)

	newStream.StreamAdmin = jwtUser
	newStream.StreamID = fmt.Sprintf("%d", utils.GenerateFnvHashID(time.Now().String()))
	logStream := log.WithFields(log.Fields{"user": jwtUser, "stream": newStream.StreamID})

	_, err := db.ArrayAppend("users", jwtUser, "streams", newStream.StreamID)
	if err != nil {
		logStream.Errorf("append stream error %s", err)
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	err = db.Add("streams", newStream)
	if err != nil {
		logStream.Errorf("add stream error %s", err)
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//subscribe to mqtt topic for this stream.
	topic := newStream.StreamAdmin + "/" + newStream.StreamID
	m.subscribe(topic)

	//all ok
	c.JSON(http.StatusCreated, gin.H{"message": newStream.StreamID, "status": http.StatusCreated})
}

//addStream adds a pre-existing public stream to a user account
func addStream(c *gin.Context) {
	db := getDB(c)
	//userID := c.Param("ID")
	jwToken := c.MustGet("jwt").(*jwt.Token)
	jwtUser := jwToken.Claims["sub"].(string)
	streamID := c.Param("STREAMID")
	stream := newStream()
	logStream := log.WithFields(log.Fields{"user": jwtUser, "stream": streamID})

	//find the stream
	_, err := db.FindById("streams", streamID, &stream)
	if err != nil {
		logStream.Errorf("find stream error %s", err)
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//if private, abort
	if !stream.StreamAccess {
		logStream.Warnf("Stream is private")
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//add stream to user account
	_, err = db.ArrayAppend("users", jwtUser, "streams", streamID)
	if err != nil {
		logStream.Errorf("append stream error %s", err)
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
	//userID := c.Param("ID")
	streamID := c.Param("STREAMID")
	stream := newStream()

	jwToken := c.MustGet("jwt").(*jwt.Token)
	jwtUser := jwToken.Claims["sub"].(string)
	logStream := log.WithFields(log.Fields{"user": jwtUser, "stream": streamID})

	//Find the stream
	_, err := db.FindById("streams", streamID, &stream)
	if err != nil {
		logStream.Errorf("find stream error %s", err)
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//owner of this stream is not logged in user
	if stream.StreamAdmin != jwtUser {
		if !stream.StreamAccess { //and if private
			logStream.Warnf("Stream is private")
			status := http.StatusNotFound
			c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
			return
		}
	}
	//}

	//render stream struct
	c.JSON(http.StatusOK, stream)
}

//getAllStreams returns all data streams for a particular user account
func getAllStreams(c *gin.Context) {
	db := getDB(c)
	//userID := c.Param("ID")
	userData := newUser()
	streamList := []stream{}

	jwToken := c.MustGet("jwt").(*jwt.Token)
	jwtUser := jwToken.Claims["sub"].(string)

	log.WithField("user", jwtUser).Debugln("getAllStreams")
	//TODO: 2 db queries can probably be merged using a join or something

	_, err := db.FindById("users", jwtUser, &userData)
	if err != nil {
		log.WithField("user", jwtUser).Errorf("find user error %s", err)
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	if len(userData.Streams) > 0 { //only search if there are streams listed
		_, err := db.FindAllById("streams", userData.Streams, &streamList)
		if err != nil {
			log.WithField("user", jwtUser).Errorf("find all streams error %s", err)
			status := http.StatusInternalServerError
			c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
			return
		}
	}

	//render streams list
	c.JSON(http.StatusOK, streamList)
}

//removeStream removes a specific data stream from current user account
//The opposite of add stream.
func removeStream(c *gin.Context) {
	db := getDB(c)
	//userID := c.Param("ID")
	streamID := c.Param("STREAMID")

	jwToken := c.MustGet("jwt").(*jwt.Token)
	jwtUser := jwToken.Claims["sub"].(string)

	_, err := db.ArrayDeleteById("users", jwtUser, "streams", streamID)
	if err != nil {
		log.WithFields(log.Fields{"user": jwtUser, "stream": streamID}).Errorf("remove stream error %s", err)
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//return deleted message
	c.Data(204, gin.MIMEJSON, nil)
	//c.JSON(http.StatusNoContent, gin.H{"message": http.StatusText(http.StatusNoContent), "status": http.StatusNoContent})
}

//deleteStream deletes a specific data stream from the database
func deleteStream(c *gin.Context) {
	db := getDB(c)
	m := getMQTT(c)
	//userID := c.Param("ID")
	streamID := c.Param("STREAMID")
	stream := newStream()

	jwToken := c.MustGet("jwt").(*jwt.Token)
	jwtUser := jwToken.Claims["sub"].(string)
	logStream := log.WithFields(log.Fields{"user": jwtUser, "stream": streamID})

	_, err := db.ArrayDeleteById("users", jwtUser, "streams", streamID)
	if err != nil {
		logStream.Errorf("array delete stream error %s", err)
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//Find the stream
	_, err = db.FindById("streams", streamID, &stream)
	if err != nil {
		logStream.Errorf("find stream error %s", err)
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//TODO: This can probably be combined in ReQL with findbyID, check ownership then delete
	if stream.StreamAdmin == jwtUser {
		err = db.DelById("streams", streamID)
		if err != nil {
			logStream.Errorf("delete stream error %s", err)
			status := http.StatusNotFound
			c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
			return
		}
	}

	//unsubscribe to mqtt topic for this stream.
	topic := stream.StreamAdmin + "/" + stream.StreamID
	m.unsubscribe(topic)

	//return deleted message
	c.Data(204, gin.MIMEJSON, nil)
	//c.JSON(http.StatusNoContent, gin.H{"message": http.StatusText(http.StatusNoContent), "status": http.StatusNoContent})
}

//createDataPoint adds a new datapoint to the db for a particular stream
//
// curl -X POST -H "Content-Type: application/json" http://localhost:8000/api/v1/users/210204461/streams/3568448099/data \
//   -H "Authorization: Bearer <token>" -d '{"value":"21"}'
func createDataPoint(c *gin.Context) {

	streamID := c.Param("STREAMID")

	db := getDB(c)
	stream := newStream()

	jwToken := c.MustGet("jwt").(*jwt.Token)
	jwtUser := jwToken.Claims["sub"].(string)

	dataPoint := newDataPoint()
	logData := log.WithFields(log.Fields{"user": jwtUser, "data": dataPoint})
	body := c.Request.Body
	log.Debugln(body)
	err := c.BindJSON(&dataPoint)
	if err != nil {
		logData.Errorf("bindJSON Error %s", err)
	}

	//TODO: Can this be combined with add? Find and Add?
	//Find the stream
	_, err = db.FindById("streams", streamID, &stream)
	if err != nil {
		logData.Errorf("find stream error %s", err)
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//can't add data if you don't own the stream
	if stream.StreamAdmin != jwtUser {
		logData.Warnln("user not authorized")
		status := http.StatusUnauthorized
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	dataPoint.StreamID = streamID

	//always need a timestamp
	nullTime := time.Time{}
	if dataPoint.TimeStamp == nullTime {
		dataPoint.TimeStamp = time.Now()
	}

	err = db.Add("dataPoints", dataPoint)
	if err != nil {
		logData.Errorf("add data error %s", err)
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
	stream := newStream()

	jwToken := c.MustGet("jwt").(*jwt.Token)
	jwtUser := jwToken.Claims["sub"].(string)
	logStream := log.WithFields(log.Fields{"user": jwtUser, "stream": streamID})

	dataPoints := []dataPoint{}

	//Find stream
	_, err := db.FindById("streams", streamID, &stream)
	if err != nil {
		logStream.Errorf("find stream error %s", err)
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//owner of this stream is not logged in user
	if stream.StreamAdmin != jwtUser {
		if !stream.StreamAccess { //and if private
			logStream.Warnln("stream is private")
			status := http.StatusNotFound
			c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
			return
		}
	}

	filter := map[string]string{"field": "streamId", "value": streamID}
	//found false continues with empty struct. Error returns error message.
	_, err = db.FindAllEq("dataPoints", filter, &dataPoints)
	if err != nil {
		logStream.Errorf("find all dataPoints error %s", err)
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//render datapoint list
	c.JSON(http.StatusOK, dataPoints)
}

//createTrigger adds a new conditional trigger to a user account
func createTrigger(c *gin.Context) {
	db := getDB(c)
	newTrigger := newTrigger()

	c.BindJSON(&newTrigger)
	newTrigger.TriggerID = fmt.Sprintf("%d", utils.GenerateFnvHashID(time.Now().String()))

	jwToken := c.MustGet("jwt").(*jwt.Token)
	jwtUser := jwToken.Claims["sub"].(string)
	//userID := c.Param("ID")

	_, err := db.ArrayAppend("users", jwtUser, "triggers", newTrigger)
	if err != nil {
		log.WithField("user", jwtUser).Errorf("array append trigger error %s", err)
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//all ok
	c.JSON(http.StatusCreated, gin.H{"message": newTrigger.TriggerID, "status": http.StatusCreated})
}

//getAllTriggers returns a list of all triggers for a user account
func getAllTriggers(c *gin.Context) {
	db := getDB(c)
	userData := newUser()

	jwToken := c.MustGet("jwt").(*jwt.Token)
	jwtUser := jwToken.Claims["sub"].(string)

	_, err := db.FindById("users", jwtUser, &userData)
	if err != nil {
		log.WithField("user", jwtUser).Errorf("get all trigger error %s", err)
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	triggerList := userData.Triggers

	//render trigger list
	c.JSON(http.StatusOK, triggerList)

}

//getTrigger changes a triggers values/settings
func getTrigger(c *gin.Context) {
	db := getDB(c)

	jwToken := c.MustGet("jwt").(*jwt.Token)
	jwtUser := jwToken.Claims["sub"].(string)

	triggerID := c.Param("TRIGGERID")
	trigger := newTrigger()

	_, err := db.ArrayFindById("users", jwtUser, "triggers", triggerID, trigger)
	if err != nil {
		log.WithFields(log.Fields{"user": jwtUser, "trigger": triggerID}).Errorf("find trigger error %s", err)
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//all ok
	c.JSON(http.StatusOK, gin.H{"message": triggerID, "status": http.StatusOK})
}

//modifyTrigger changes a triggers values/settings
func modTrigger(c *gin.Context) {
	db := getDB(c)

	jwToken := c.MustGet("jwt").(*jwt.Token)
	jwtUser := jwToken.Claims["sub"].(string)

	triggerID := c.Param("TRIGGERID")
	trigger := newTrigger()
	c.BindJSON(&trigger)

	//TODO: find trigger in user trigger array. Replace it.
	_, err := db.ArrayAppend("users", jwtUser, "triggers", trigger)
	if err != nil {
		log.WithFields(log.Fields{"user": jwtUser, "trigger": triggerID}).Errorf("mod trigger error %s", err)
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//all ok
	c.JSON(http.StatusOK, gin.H{"message": triggerID, "status": http.StatusOK})
}

//deleteTrigger removes a pre-existing trigger from a user account
func deleteTrigger(c *gin.Context) {
	db := getDB(c)

	jwToken := c.MustGet("jwt").(*jwt.Token)
	jwtUser := jwToken.Claims["sub"].(string)

	triggerID := c.Param("TRIGGERID")

	_, err := db.ArrayDeleteById("users", jwtUser, "triggers", triggerID)
	if err != nil {
		log.WithFields(log.Fields{"user": jwtUser, "trigger": triggerID}).Errorf("delete trigger error %s", err)
		status := http.StatusInternalServerError
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//return deleted message
	//c.JSON(http.StatusNoContent, gin.H{"message": http.StatusText(http.StatusNoContent), "status": http.StatusNoContent})
	c.Data(http.StatusNoContent, gin.MIMEJSON, nil)

}

//handleWebSocket handles incoming requests to setup a websocket
//and return live data updates for a particular stream
func handleWebSocket(c *gin.Context) {

	db := getDB(c)
	streamID := c.Param("STREAMID")
	stream := newStream()

	jwToken := c.MustGet("jwt").(*jwt.Token)
	jwtUser := jwToken.Claims["sub"].(string)
	logStream := log.WithFields(log.Fields{"user": jwtUser, "stream": streamID})

	dataChan := make(chan interface{}, 256)

	//Find stream
	_, err := db.FindById("streams", streamID, &stream)
	if err != nil {
		logStream.Errorf("find stream error %s", err)
		status := http.StatusNotFound
		c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
		return
	}

	//owner of this stream is not logged in user
	if stream.StreamAdmin != jwtUser {
		if !stream.StreamAccess { //and if private
			logStream.Warnln("stream is private")
			status := http.StatusNotFound
			c.JSON(status, gin.H{"message": http.StatusText(status), "status": status})
			return
		}
	}

	//upgrade to websocket connection
	ws, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logStream.Errorf("Failed to set websocket upgrade %s", err)
		return
	}
	defer ws.Close()

	go reader(ws)

	go writer(ws, dataChan) //recieve data from feedData channel ping until data recieved

	feedData, err := db.ChangesByIdx("dataPoints", "streamId", streamID, 1) //TODO: Filter per stream or user
	if err != nil {
		logStream.Errorf("change feed error %s", err)
	}
	defer feedData.Close()

	var data dataPoint

	for feedData.Next(&data) { //blocks forever

		dataChan <- data
		//TODO: write to channel

	}
	if feedData.Err() != nil {
		logStream.Errorf("feedData error %s", feedData.Err())
	}

}

func writer(ws *websocket.Conn, dataChan <-chan interface{}) {

	writeWait := time.Second * 10
	pingPeriod := ((time.Second * 60) * 9) / 10
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		ws.Close()
	}()

	for {
		select {
		case message := <-dataChan:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			err := ws.WriteJSON(message)
			if err != nil {
				log.Errorf("write JSON error %s", err.Error())
				return
			}
		case <-ticker.C:
			err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait))
			if err != nil {
				log.Errorf("write ping error %s", err.Error())
				return
			}
		}
	}

}

//reader is used to read from a websocket
func reader(ws *websocket.Conn) {
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(time.Second * 60))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(time.Second * 60)); return nil })

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Errorf("read msg error %s", err.Error())
			break
		}
	}
}

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
