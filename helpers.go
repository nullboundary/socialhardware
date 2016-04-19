package main

import (
	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	jwt_lib "github.com/dgrijalva/jwt-go"
	"github.com/nullboundary/utilbelt"

	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//createJWToken creates a new jwt token to be returned usually during login
func createJWToken(tokenName string, signKey []byte, subClaim string) (map[string]string, error) {

	token := jwt_lib.New(jwt_lib.GetSigningMethod("HS256"))
	// Set some claims
	token.Claims["sub"] = subClaim
	token.Claims["iat"] = time.Now().Unix()
	token.Claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	// Sign and get the complete encoded token as a string
	tokenString, err := token.SignedString(signKey)
	if err != nil {
		return nil, err
	}

	return map[string]string{tokenName: tokenString}, nil
}

func createRawURL(token string, userEmail string, expires string, requestHost string) string {

	u := url.URL{}
	u.Scheme = "http"
	u.Host = requestHost
	u.Path = "users/verify"
	q := u.Query()
	q.Add("email", userEmail)
	q.Add("expires", expires)

	q.Add("token", token)
	u.RawQuery = q.Encode()

	return u.String()

}

//checkTriggers check a list of triggers matching this stream, see if they need triggering
func checkTriggers(db *storer.ReThink, data *dataPoint) {

	//Get list of triggers matching this stream
	go func() { //complete in goroutine so it doesn't slow down request

		log.Println("[DEBUG] checkTriggers")

		triggerList := []trigger{}
		_, err := db.FindAllByArrayItem("users", "triggers", "streamId", data.StreamID, true, &triggerList) //TODO: cache this or something
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return
		}

		for _, trigger := range triggerList {

			//log.Printf("ID: %s Trigger: if %s then %s \n", trigger.TriggerID, trigger.TriggerCondition, trigger.Triggerhook.URL)

			if triggerMatch(trigger, data) {
				//every client request in a seperate goroutine (some could be slow to return)
				go func() {

					u, err := url.Parse(trigger.URL)
					if err != nil {
						log.Printf("[ERROR] %s", err)
						return
					}
					//TODO: maybe localhost shouldn't be allowed after deployment ?
					if u.Scheme == "localhost" {
						u.Scheme = "http://localhost"
					}

					log.Println("URL: ", u)

					//if trigger.Body
					bytesBody, err := trigger.Body.MarshalJSON()
					if err != nil {
						log.Printf("[ERROR] %s", err)
					}
					bodyReader := bytes.NewReader(bytesBody)

					//var jsonStr = []byte(`{"title":"Buy cheese and bread for breakfast."}`)
					req, err := http.NewRequest(trigger.Method, u.String(), bodyReader)
					req.Header.Set("Authorization", "0e1766cb9a519d3")

					header := strings.Split(trigger.Headers, ":")
					headerKey := strings.TrimSpace(header[0])
					headerValue := strings.TrimSpace(header[1])
					req.Header.Add(headerKey, headerValue)

					client := &http.Client{}
					resp, err := client.Do(req)
					if err != nil {
						log.Printf("[ERROR] %s", err)
						return
					}
					defer resp.Body.Close()

					log.Println("response Status:", resp.Status)
					log.Println("response Headers:", resp.Header)
					body, _ := ioutil.ReadAll(resp.Body)
					log.Println("response Body:", string(body))
				}()
			}

		}

	}()

}

func triggerMatch(t trigger, data *dataPoint) bool {

	switch t.CondExpr {
	case "eq":

		trigCond, err := strconv.ParseFloat(t.CondValue, 64)
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return false
		}

		value, err := data.Value.valueToFloat()
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return false
		}
		return value == trigCond

	case "gt":

		trigCond, err := strconv.ParseFloat(t.CondValue, 64)
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return false
		}

		value, err := data.Value.valueToFloat()
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return false
		}
		return value > trigCond

	case "lt":

		trigCond, err := strconv.ParseFloat(t.CondValue, 64)
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return false
		}

		value, err := data.Value.valueToFloat()
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return false
		}
		return value < trigCond

	case "regex":
		condRegex, err := regexp.Compile(t.CondValue)
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return false
		}

		value, err := data.Value.valueToString()
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return false
		}

		return condRegex.MatchString(value)

	default: //default (no type) is ==.

		trigCond, err := strconv.ParseFloat(t.CondValue, 64)
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return false
		}

		value, err := data.Value.valueToFloat()
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return false
		}
		return value == trigCond
	}

}

func (v value) valueToString() (string, error) {

	if v.ValueString != "" {
		return v.ValueString, nil
	} else if v.ValueInt != int64(0) {
		return strconv.FormatInt(v.ValueInt, 10), nil
	} else {
		return strconv.FormatFloat(v.ValueFloat, 'f', -1, 64), nil
	}
}

func (v value) valueToFloat() (float64, error) {

	if v.ValueString != "" {
		return strconv.ParseFloat(v.ValueString, 64)
	} else if v.ValueInt != int64(0) {
		return float64(v.ValueInt), nil
	} else {
		return v.ValueFloat, nil
	}
}

type mqttClient struct {
	client         *MQTT.Client
	brokerAddr     string              //the address of the MQTT broker
	clientID       string              //how this client identifies itself to the broker
	receiveHandler MQTT.MessageHandler //function to handle recieved messsages
	defaultQOS     byte                //The recieve QOS from the broker
}

func newMqttClient() *mqttClient {
	return &mqttClient{}
}

//setup configuring the mqtt client and connects to the broker
func (m *mqttClient) setup(clientID string, brokerAddr string, qos byte, db *storer.ReThink) {

	m.brokerAddr = brokerAddr
	m.clientID = clientID
	m.defaultQOS = qos

	//create a jwt token to login the api server to the mqtt broker using jwt.
	jwt, err := createJWToken("token", []byte(jWTokenKey), "socialhardware.io")
	if err != nil {
		log.Fatalf("[ERROR] create token error")
	}
	//create a ClientOptions struct setting the broker address, clientid, turn
	//off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker(m.brokerAddr)
	opts.SetClientID(m.clientID)
	opts.SetUsername("socialhardware.io")
	opts.SetPassword(jwt["token"])
	//opts.SetDefaultPublishHandler(f)

	//create and start a client using the above ClientOptions
	m.client = MQTT.NewClient(opts)

	if token := m.client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("[ERROR] %s", token.Error())
	}

	//define a function for the default message handler
	m.receiveHandler = func(client *MQTT.Client, msg MQTT.Message) {
		log.Printf("TOPIC: %s\n", msg.Topic())
		log.Printf("MSG: %s\n", msg.Payload())

		topics := strings.Split(msg.Topic(), "/")

		dataPoint := newDataPoint()

		if err := json.Unmarshal(msg.Payload(), &dataPoint); err != nil {
			log.Printf("jsonRead Error: %v", err)
			return
		}

		if len(topics) != 2 {
			log.Println("[ERROR] topic length incorrect")
			return
		}
		dataPoint.StreamID = topics[1]

		//always need a timestamp
		nullTime := time.Time{}
		if dataPoint.TimeStamp == nullTime {
			dataPoint.TimeStamp = time.Now()
		}
		log.Printf("timestamp: %s", dataPoint.TimeStamp.String())

		err := db.Add("dataPoints", dataPoint)
		if err != nil {
			log.Printf("[ERROR] %s", err.Error())
			return
		}

		checkTriggers(db, dataPoint)

	}

}

func (m *mqttClient) registerStreams(db *storer.ReThink) {

	log.Println("Register all streams with mqtt...")

	streamList := []stream{}

	_, err := db.GetAll("streams", &streamList)
	if err != nil {
		log.Fatalf("[ERROR] %s", err.Error())
		return
	}

	log.Printf("Server registering %d streams", len(streamList))

	for _, stream := range streamList {
		//subscribe to mqtt topic for this stream.
		topic := stream.StreamAdmin + "/" + stream.StreamID
		m.subscribe(topic)
	}

}

func (m *mqttClient) subscribe(topic string) {
	//subscribe to the topic /go-mqtt/sample and request messages to be delivered
	//at a maximum qos of zero, wait for the receipt to confirm the subscription
	if token := m.client.Subscribe(topic, m.defaultQOS, m.receiveHandler); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
	log.Printf("MQTT Client Subscribed to: %s", topic)
}

func (m *mqttClient) unsubscribe(topic string) {
	if token := m.client.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
	log.Printf("MQTT Client Unsubscribed to: %s", topic)
}
