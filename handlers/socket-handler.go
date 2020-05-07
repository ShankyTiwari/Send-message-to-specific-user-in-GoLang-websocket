package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// UserStruct is used for sending users with socket id
type UserStruct struct {
	Username string `json:"username"`
	SocketID string `json:"socketID"`
}

// SocketEventStruct struct of socket events
type SocketEventStruct struct {
	EventName    string      `json:"eventName"`
	EventPayload interface{} `json:"eventPayload"`
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	webSocketConnection *websocket.Conn

	// Buffered channel of outbound messages.
	send chan SocketEventStruct

	// username of the client
	username string

	// user id of the client
	userID string
}

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

func unRegisterAndCloseConnection(c *Client) {
	// Here we are unregistering the client
	c.hub.unregister <- c
	c.webSocketConnection.Close()
}

func setSocketPayloadReadConfig(c *Client) {
	// SetReadLimit sets the maximum size in bytes for a message read from the peer. If a
	// message exceeds the limit, the connection sends a close message to the peer
	// and returns ErrReadLimit to the application.
	c.webSocketConnection.SetReadLimit(maxMessageSize)

	// SetReadDeadline sets the read deadline on the underlying network connection.
	// After a read has timed out, the websocket connection state is corrupt and
	// all future reads will return an error. A zero value for t means reads will not time out.
	c.webSocketConnection.SetReadDeadline(time.Now().Add(pongWait))

	// SetPongHandler sets the handler for pong messages received from the peer.
	// The appData argument to h is the PONG message application data. The default pong handler does nothing.
	c.webSocketConnection.SetPongHandler(func(string) error { c.webSocketConnection.SetReadDeadline(time.Now().Add(pongWait)); return nil })
}

func handleSocketPayloadEvents(c *Client, socketEventPayload SocketEventStruct) {
	var socketEventResponse SocketEventStruct
	switch socketEventPayload.EventName {
	case "message":
		socketEventResponse.EventName = "message response"
		socketEventResponse.EventPayload = map[string]interface{}{
			"username": c.username,
			"message":  socketEventPayload.EventPayload.(map[string]interface{})["message"],
			"socketID": socketEventPayload.EventPayload.(map[string]interface{})["socketID"],
		}

		// More event can come here along with their logic

		reqBodyBytes := new(bytes.Buffer)
		json.NewEncoder(reqBodyBytes).Encode(socketEventResponse)

		c.hub.specificEmit <- socketEventResponse
	}
}

func getAllConnectedUsers(hub *Hub) []UserStruct {
	var users []UserStruct
	for singleClient := range hub.clients {
		users = append(users, UserStruct{
			Username: singleClient.username,
			SocketID: singleClient.userID,
		})
	}
	return users
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	var socketEventPayload SocketEventStruct

	// Unregistering the client and closing the connection
	defer unRegisterAndCloseConnection(c)

	// Setting up the Payload configuration
	setSocketPayloadReadConfig(c)

	for {
		// ReadMessage is a helper method for getting a reader using NextReader and reading from that reader to a buffer.
		_, payload, err := c.webSocketConnection.ReadMessage()

		decoder := json.NewDecoder(bytes.NewReader(payload))
		decoderErr := decoder.Decode(&socketEventPayload)

		if decoderErr != nil {
			log.Printf("error: %v", decoderErr)
			break
		}

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error ===: %v", err)
			}
			break
		}

		//  Getting the proper Payload to send the client
		handleSocketPayloadEvents(c, socketEventPayload)
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
		c.webSocketConnection.Close()
	}()
	for {
		select {
		case payload, ok := <-c.send:

			reqBodyBytes := new(bytes.Buffer)
			json.NewEncoder(reqBodyBytes).Encode(payload)
			finalPayload := reqBodyBytes.Bytes()

			// SetWriteDeadline sets the write deadline on the underlying network
			// connection. After a write has timed out, the websocket state is corrupt and
			// all future writes will return an error. A zero value for t means writes will
			// not time out.
			c.webSocketConnection.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// WriteMessage is a helper method for getting a writer using NextWriter, here closing the writer.
				c.webSocketConnection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// NextWriter returns a writer for the next message to send.
			w, err := c.webSocketConnection.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			// Write writes the message
			w.Write(finalPayload)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				json.NewEncoder(reqBodyBytes).Encode(<-c.send)
				w.Write(reqBodyBytes.Bytes())
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.webSocketConnection.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.webSocketConnection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// CreateNewSocketUser creates a new socket user
func CreateNewSocketUser(hub *Hub, connection *websocket.Conn, username string) {
	// Creating a new socket client

	uniqueID := uuid.New()
	client := &Client{
		hub:                 hub,
		webSocketConnection: connection,
		send:                make(chan SocketEventStruct),
		username:            username,
		userID:              uniqueID.String(),
	}

	// Allow collection of memory referenced by the caller by doing all work in new goroutines.
	go client.writePump()
	go client.readPump()

	// Registering the newly created client using Hub
	client.hub.register <- client
}

// HandleUserJoinEvent will handle the Join event for New socket users
func HandleUserJoinEvent(hub *Hub, client *Client) {
	hub.clients[client] = true
	BroadcastSocketEventToAllClient(hub, SocketEventStruct{
		EventName:    "join",
		EventPayload: getAllConnectedUsers(hub),
	})
}

// HandleUserDisconnectEvent will handle the Disconnect event for socket users
func HandleUserDisconnectEvent(hub *Hub, client *Client) {
	_, ok := hub.clients[client]
	if ok {
		delete(hub.clients, client)
		close(client.send)

		BroadcastSocketEventToAllClient(hub, SocketEventStruct{
			EventName:    "disconnect",
			EventPayload: getAllConnectedUsers(hub),
		})
	}
}

// EmitToSpecificClient will emit the socket event to specific socket user
func EmitToSpecificClient(hub *Hub, payload SocketEventStruct) {
	for client := range hub.clients {
		if client.userID == payload.EventPayload.(map[string]interface{})["socketID"] {
			select {
			case client.send <- payload:
			default:
				close(client.send)
				delete(hub.clients, client)
			}
		}
	}
}

// BroadcastSocketEventToAllClient will emit the socket events to all socket users
func BroadcastSocketEventToAllClient(hub *Hub, payload SocketEventStruct) {
	for client := range hub.clients {
		select {
		case client.send <- payload:
		default:
			close(client.send)
			delete(hub.clients, client)
		}
	}
}
