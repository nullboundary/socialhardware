package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/emicklei/forest"
)

var shw = forest.NewClient("http://localhost:8000", new(http.Client))
var testJwToken string
var testUserID string
var testStreamID string
var testTriggerID string

// auth/login

func TestBadLogin(t *testing.T) {
	cfg := forest.NewConfig("/auth/login").
		Header("Accept", "application/json").
		Body(`{"email": "test@gmail.com","password": "11111"}`)
	r := shw.POST(t, cfg)
	forest.ExpectStatus(t, r, 401)

}

func TestGoodLogin(t *testing.T) {
	cfg := forest.NewConfig("/auth/login").
		Header("Accept", "application/json").
		Body(`{"email": "testuser@gmail.com","password": "testminnow"}`)
	r := shw.POST(t, cfg)
	forest.ExpectStatus(t, r, 200)
	forest.ExpectJSONHash(t, r, func(hash map[string]interface{}) {
		if hash["user"] != "1524579473" {
			t.Error("user id incorrect")
		}
		testUserID = hash["user"].(string)
		fmt.Printf("%v \n", hash["jwt"])
		j := hash["jwt"].(map[string]interface{})
		testJwToken = j["token"].(string)
		fmt.Println(testJwToken)
	})

}

// api/v1/users/{id}
func TestGetUser(t *testing.T) {
	bearer := "Bearer " + testJwToken
	cfg := forest.NewConfig("/api/v1/users").
		Header("Accept", "application/json").
		Header("Authorization", bearer)
	r := shw.GET(t, cfg)
	forest.ExpectStatus(t, r, 200)

}

func TestGetUserBadAuth(t *testing.T) {
	bearer := "Bearer " + "k39dk.jidiww.399f"
	cfg := forest.NewConfig("/api/v1/users").
		Header("Accept", "application/json").
		Header("Authorization", bearer)
	r := shw.GET(t, cfg)
	forest.ExpectStatus(t, r, 401)

}

// api/v1/users/{id}
func BenchmarkGetUser(b *testing.B) {
	//rCount := 0
	for n := 0; n < b.N; n++ {
		bearer := "Bearer " + testJwToken
		cfg := forest.NewConfig("/api/v1/users").
			Header("Accept", "application/json").
			Header("Authorization", bearer)
		r := shw.GET(b, cfg)
		forest.ExpectStatus(b, r, 200)

	}

}

// api/v1/streams/{id}

func TestCreateStream(t *testing.T) {
	bearer := "Bearer " + testJwToken
	cfg := forest.NewConfig("/api/v1/streams/").
		Header("Accept", "application/json").
		Header("Authorization", bearer).
		Body(`{"streamName": "teststreamgo", "streamDesc": "go testing stream", "streamLoc": "here"}`)
	r := shw.POST(t, cfg)
	forest.ExpectStatus(t, r, 201)
	forest.ExpectJSONHash(t, r, func(hash map[string]interface{}) {

		testStreamID = hash["message"].(string)
	})

}

func TestGetStream(t *testing.T) {
	bearer := "Bearer " + testJwToken
	cfg := forest.NewConfig("/api/v1/streams/{ID}", testStreamID).
		Header("Accept", "application/json").
		Header("Authorization", bearer)
	r := shw.GET(t, cfg)
	forest.ExpectStatus(t, r, 200)

}

func TestDeleteStream(t *testing.T) {
	bearer := "Bearer " + testJwToken
	cfg := forest.NewConfig("/api/v1/streams/{ID}", testStreamID).
		Header("Accept", "application/json").
		Header("Authorization", bearer)
	r := shw.DELETE(t, cfg)
	forest.ExpectStatus(t, r, 204)

}

// api/v1/streams

func TestGetAllStreams(t *testing.T) {
	bearer := "Bearer " + testJwToken
	cfg := forest.NewConfig("/api/v1/streams").
		Header("Accept", "application/json").
		Header("Authorization", bearer)
	r := shw.GET(t, cfg)
	forest.ExpectStatus(t, r, 200)

}

// api/v1/triggers

func TestCreateTrigger(t *testing.T) {
	bearer := "Bearer " + testJwToken
	cfg := forest.NewConfig("/api/v1/triggers/").
		Header("Accept", "application/json").
		Header("Authorization", bearer).
		Body(`{"streamId": "68826694", "condExpr":  "gt" , "condValue":  "20" ,"method":  "GET" ,"url":  "localhost:8000/ping"}`)
	r := shw.POST(t, cfg)
	forest.ExpectStatus(t, r, 201)
	forest.ExpectJSONHash(t, r, func(hash map[string]interface{}) {

		testTriggerID = hash["message"].(string)
	})

}

func TestGetAllTriggers(t *testing.T) {
	bearer := "Bearer " + testJwToken
	cfg := forest.NewConfig("/api/v1/triggers/").
		Header("Accept", "application/json").
		Header("Authorization", bearer)
	r := shw.GET(t, cfg)
	forest.ExpectStatus(t, r, 200)

}

func TestDeleteTrigger(t *testing.T) {
	bearer := "Bearer " + testJwToken
	cfg := forest.NewConfig("/api/v1/triggers/{ID}", testTriggerID).
		Header("Accept", "application/json").
		Header("Authorization", bearer)
	r := shw.DELETE(t, cfg)
	forest.ExpectStatus(t, r, 204)

}
