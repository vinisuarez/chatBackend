package main

import (
	"bytes"
	"chatBackend/auth"
	"chatBackend/pb"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(*http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	conn   *websocket.Conn
	server *Server
	send   chan []byte
	id     uuid.UUID
	name   string
}

func newClient(conn *websocket.Conn, server *Server, id string, name string) *Client {

	client := &Client{
		server: server,
		conn:   conn,
		send:   make(chan []byte, 256),
		name:   name,
	}
	if id != "" {
		client.id, _ = uuid.Parse(id)
	}
	return client
}

const listOnline string = "!online"

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.server.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		stringMessage := string(bytes.TrimSpace(bytes.Replace(message, newline, space, -1)))

		if stringMessage == listOnline {
			c.server.listOnline <- c
		} else {
			msg := &pb.ChatMessage{
				FromUser: c.name,
				SentAt:   timestamppb.Now(),
				Text:     string(message),
			}
			c.server.broadcast <- msg
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func ServeWs(server *Server, w http.ResponseWriter, r *http.Request) {
	userCtxValue := r.Context().Value(auth.UserContextKey)
	if userCtxValue == nil {
		log.Println("Not authenticated")
		return
	}

	user := userCtxValue.(*pb.User)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := newClient(conn, server, user.Id, user.Name)

	go client.writePump()
	go client.readPump()

	server.register <- client
}
