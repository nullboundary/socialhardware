package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"bitbucket.org/cicadaDev/storer"
	"code.google.com/p/go.crypto/bcrypt"
	"github.com/dancannon/gorethink/encoding"
)

type user struct {
	ID        string    `json:"id" gorethink:"id"`                          //A unique ID for each user
	Email     string    `json:"email" gorethink:"email" binding:"required"` //the users email works as an id
	PassCrypt []byte    `json:"passcrypt,omitempty" gorethink:"passcrypt"`  //bcrypted password byte slice
	Created   time.Time `json:"created,omitempty" gorethink:"created"`      //Date account was created
	Verified  bool      `json:"verified,omitempty" gorethink:"verified"`    //is the account email verified?
	Streams   []string  `json:"streams,omitempty" gorethink:"streams"`      //list of stream ids
	Triggers  []trigger `json:"triggers,omitempty" gorethink:"triggers"`
}

//a stream represents a single stream of data from 1 stream
type stream struct {
	StreamAdmin  string    `json:"streamAdmin,omitempty" gorethink:"streamAdmin,omitempty"`   //owner of the stream
	StreamID     string    `json:"id" gorethink:"id"`                                         //A unique ID for each sensor sending data
	StreamName   string    `json:"streamName,omitempty" gorethink:"streamName,omitempty"`     //A human readable name
	StreamType   string    `json:"streamType,omitempty" gorethink:"streamType,omitempty"`     //The type of stream/sensor. Example: TGS2620 is a model of VOC air sensor
	StreamDesc   string    `json:"streamDesc,omitempty" gorethink:"streamDesc,omitempty"`     //A wordy description of the particular data stream
	StreamAccess bool      `json:"streamAccess,omitempty" gorethink:"streamAccess,omitempty"` //true = public, false = private
	StreamLoc    *location `json:"streamLoc,omitempty" gorethink:"streamLoc,omitempty"`       //the lat long location of a stream
	StreamTags   []string  `json:"streamTags,omitempty" gorethink:"streamTags,omitempty"`     //list of stream description tags
}

type trigger struct {
	StreamID  string          `json:"streamId" gorethink:"streamId"`                   //A unique ID for each sensor sending data
	TriggerID string          `json:"triggerId,omitempty" gorethink:"triggerId"`       //A unique ID for each sensor sending data
	CondValue string          `json:"condValue" gorethink:"condValue"`                 //a conditional value
	CondExpr  string          `json:"condExpr" gorethink:"condExpr"`                   //a conditional expression >= etc.
	URL       string          `json:"url,omitempty" gorethink:"url,omitempty"`         //A url to send a request to if triggered
	Method    string          `json:"method,omitempty" gorethink:"method,omitempty"`   //request method GET, POST, PUT, etc
	Headers   string          `json:"headers,omitempty" gorethink:"headers,omitempty"` //HTTP headers to add to request
	Body      json.RawMessage `json:"body,omitempty" gorethink:"body,omitempty"`       //raw json request input body for PUT, POST requests
}

type webhook struct {
	URL     string      `json:"url,omitempty" gorethink:"url,omitempty"`         //A url to send a request to if triggered
	Method  string      `json:"method,omitempty" gorethink:"method,omitempty"`   //GET, POST, PUT, etc
	Headers http.Header `json:"headers,omitempty" gorethink:"headers,omitempty"` //HTTP headers to add to request
	Form    url.Values  `json:"form,omitempty" gorethink:"form,omitempty"`       //Form values
}

//Location provides information about a stream location.
type location struct {
	Altitude  float64 `json:"altitude,omitempty" gorethink:"altitude,omitempty"`                     //Altitude, in meters, of the location.
	Latitude  float64 `json:"latitude,omitempty" gorethink:"latitude,omitempty" valid:"latitude"`    //Latitude, in degrees, of the location.
	Longitude float64 `json:"longitude,omitempty" gorethink:"longitude,omitempty" valid:"longitude"` //Longitude, in degrees, of the location.
}

//dataPoint is the basic unit data recording
type dataPoint struct {
	id        string    `json:"-" gorethink:"id"`
	StreamID  string    `json:"-" gorethink:"streamId"`
	TimeStamp time.Time `json:"timeStamp,omitempty" gorethink:"timestamp"`         //time
	Value     *value    `json:"value,omitempty" gorethink:"value,omitempty"`       //The data value
	metaInfo  string    `json:"metaInfo,omitempty" gorethink:"metaInfo,omitempty"` //A note or extra info about the data point
	loc       *location `json:"loc,omitempty" gorethink:"loc,omitempty"`           //the lat long location of a recorded value

}

//value is used with json.Unmarshal interface to assign either string, float or int to value
type value struct {
	ValueInt    int64   `json:"valueint,omitempty" gorethink:"valueint,omitempty"`
	ValueFloat  float64 `json:"valuefloat,omitempty" gorethink:"valuefloat,omitempty"`
	ValueString string  `json:"valuestring,omitempty" gorethink:"valuestring,omitempty" valid:"msg"`
}

func newUser() *user {
	return &user{}
}

//	newStream returns a pointer to a stream struct.
func newStream() *stream {
	return &stream{}
}

//newTrigger returns a pointer to a trigger struct
func newTrigger() *trigger {
	return &trigger{}
}

//	newDataPoint returns a pointer to a dataPoint struct.
func newDataPoint() *dataPoint {
	return &dataPoint{}
}

func (u *user) setPassword(password string) {

	//bcrypt password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12) //cost ~400ms on mac-air
	if err != nil {
		log.Println(err)
	}
	u.PassCrypt = hashedPassword

}

//	MarshalRQL takes value type data input to rethinkdb which is marshaled into sub values of
//	int, float or string
func (v *value) MarshalRQL() (interface{}, error) {
	log.Printf("[DEBUG] marshalRQL  %+v", v)

	if v.ValueInt != int64(0) {
		return encoding.Encode(v.ValueInt)
	}
	if v.ValueFloat != float64(0) {
		return encoding.Encode(v.ValueFloat)
	}
	if v.ValueString != "" {
		return encoding.Encode(v.ValueString)
	}

	return encoding.Encode(nil)
}

//	UnmarshalRQL takes value type data output from rethinkdb which is unmarshaled into a sub value of
//	int, float or string
func (v *value) UnmarshalRQL(b interface{}) error {

	log.Printf("[DEBUG] unmarshalRQL %+v", b)
	s := ""
	//decode to string first
	if err := encoding.Decode(&s, b); err == nil {
		log.Printf("[DEBUG] %T - %s\n", s, s)
	}
	//try parse string as int
	if n, err := strconv.ParseInt(s, 10, 64); err == nil { //FIXME: for "0131" creates "131"
		log.Printf("[DEBUG] %T - %d\n", n, n)
		v.ValueInt = n
		return nil
	}
	//try parse string as float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		log.Printf("[DEBUG] %T - %f\n", f, f)
		v.ValueFloat = f
		return nil
	}
	f := `2006-01-02T15:04:05.000Z` //convert ISO Z to +- tZ
	if tt, err := time.Parse(f, s); err == nil {
		log.Printf("[DEBUG] %T - %s\n", tt, tt.String())
		iso8601 := "2006-01-02T15:04:05-07:00" //W3C
		s := tt.Format(iso8601)
		v.ValueString = s
		return nil
	}

	v.ValueString = s

	return nil

}

//compiler check to make sure value is valid
var _ storer.RtValue = (*value)(nil)

//	UnmarshalJSON handles json unmarshalling from doc to struct of Value types for fields.
func (v *value) UnmarshalJSON(b []byte) (err error) {
	n, f, s := int64(0), float64(0), ""
	log.Printf("[DEBUG] unmarshalJSON")
	if err = json.Unmarshal(b, &s); err == nil {
		v.ValueString = s
		return
	}
	if err = json.Unmarshal(b, &f); err == nil {
		v.ValueFloat = f
		return
	}
	if err = json.Unmarshal(b, &n); err == nil {
		v.ValueInt = n

	}

	return
}

//	MarshalJSON handles json marshalling from struct to doc of Value types for fields.
func (v *value) MarshalJSON() ([]byte, error) {

	if v.ValueInt != 0 {
		return json.Marshal(v.ValueInt)
	}
	if v.ValueFloat != float64(0) {
		return json.Marshal(v.ValueFloat)
	}
	if v.ValueString != "" {
		return json.Marshal(v.ValueString)
	}
	return json.Marshal(nil)
}
