package main

/*
import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)


type hub struct {
	// Registered connections.
	connections map[*connection]bool

	// Inbound messages from the connections.
	broadcast chan interface{}

	// Register requests from the connections.
	register chan *connection

	// Unregister requests from connections.
	unregister chan *connection
}

type connection struct {
	// The websocket connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	send chan interface{}
}


func handleWebSocket(c *gin.Context) {

	db := getDB(c)
	//streamID := c.Param("STREAMID")

	conn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println("Failed to set websocket upgrade: %+v", err)
		return
	}
	for {
		//feedData, err := db.ChangesById("dataPoints", streamID, 2)
		feedData, err := db.Changes("dataPoints", 2)
		if err != nil {
			fmt.Printf("[ERROR] %s", err)
		}

		var data dataPoint
		for feedData.Next(&data) {

			fmt.Printf("%v", data)

			//msgType, msg, err := conn.ReadMessage()
			//if err != nil {
			//	break
			//}
			conn.WriteJSON(data)
			//conn.WriteMessage(msgType, msg)
		}
	}
}

func handleWebSocket(c *gin.Context) {
	db := getDB(c)
	newChangesHandler(db.ChangesChan)
}

func newChangesHandler(fn func(ch chan interface{}, tableName string, interval int)) http.HandlerFunc {
	h := newHub()
	go h.run()

	fn(h.broadcast, "dataPoints", 2)

	return wsHandler(h)
}

func wsHandler(h hub) http.HandlerFunc {
	log.Println("Starting websocket server")
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsupgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println("Failed to set websocket upgrade: %+v", err)
			return
		}
		c := &connection{send: make(chan interface{}, 256), ws: conn}
		h.register <- c
		defer func() { h.unregister <- c }()
		go c.writer()
		c.reader()
	}
}

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (c *connection) reader() {
	for {
		_, _, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
	}
	c.ws.Close()
}

func (c *connection) writer() {
	for change := range c.send {
		err := c.ws.WriteJSON(change)
		if err != nil {
			break
		}
	}
	c.ws.Close()
}

func newHub() hub {
	return hub{
		broadcast:   make(chan interface{}),
		register:    make(chan *connection),
		unregister:  make(chan *connection),
		connections: make(map[*connection]bool),
	}
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.connections[c] = true
		case c := <-h.unregister:
			if _, ok := h.connections[c]; ok {
				delete(h.connections, c)
				close(c.send)
			}
		case m := <-h.broadcast:
			for c := range h.connections {
				select {
				case c.send <- m:
				default:
					delete(h.connections, c)
					close(c.send)
				}
			}
		}
	}
}
*/
