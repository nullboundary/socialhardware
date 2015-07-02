package main

import (
	"bitbucket.org/cicadaDev/storer"
	"bitbucket.org/cicadaDev/utils"
	jwt_lib "github.com/dgrijalva/jwt-go"

	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var tokenPrivateKey = []byte(`e&?kz,qIUnDWpnSuB_yr[vAEE}Z_(YU`) //TODO: lets make a new key and put this somewhere safer!

func accessToken(authString string, seeds ...string) bool {

	//1. Verify that the authentication token is correct.
	header := strings.Split(authString, " ")
	if len(header) < 2 {
		return false //header is malformed
	}
	authToken := header[1] //take the second value "api-key token"
	if ok, err := utils.VerifyToken(tokenPrivateKey, authToken, seeds...); !ok {

		if err != nil {
			utils.Check(err)
		}
		return false //verify failed
	}

	return true

}

func createJWToken(tokenName string, signKey []byte, subClaim string) (map[string]string, error) {

	token := jwt_lib.New(jwt_lib.GetSigningMethod("HS256"))
	// Set some claims
	token.Claims["sub"] = subClaim
	//token.Claims["iss"] = "https://socialhardware.io"
	//token.Claims["scopes"] = ["explorer", "solar-harvester", "seller"]
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
	u.Scheme = "https"
	u.Host = requestHost
	u.Path = "verify"
	q := u.Query()
	q.Add("email", userEmail)
	q.Add("expires", expires)

	q.Add("token", token)
	u.RawQuery = q.Encode()

	return u.String()

}

func checkTriggers(db *storer.ReThink, data *dataPoint) {

	//Get list of triggers matching this stream
	go func() { //complete in goroutine so it doesn't slow down request

		log.Println("[DEBUG] checkTriggers")

		triggerList := []trigger{}
		_, err := db.FindAllByArrayItem("users", "triggers", "streamId", data.StreamID, &triggerList) //TODO: cache this or something
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
					bytesBody, _ := trigger.Body.MarshalJSON()
					bodyReader := bytes.NewReader(bytesBody)

					//var jsonStr = []byte(`{"title":"Buy cheese and bread for breakfast."}`)
					req, err := http.NewRequest(trigger.Method, u.String(), bodyReader)
					//req.Header.Set("Content-Type", "application/json")

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
