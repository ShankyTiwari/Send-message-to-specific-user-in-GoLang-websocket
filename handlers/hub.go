package handlers

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan SocketEventStruct

	// Inbound messages from the clients.
	specificEmit chan SocketEventStruct

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

// NewHub will will give an instance of an Hub
func NewHub() *Hub {
	return &Hub{
		broadcast:    make(chan SocketEventStruct),
		specificEmit: make(chan SocketEventStruct),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		clients:      make(map[*Client]bool),
	}
}

// Run will execute Go Routines to check incoming Socket events
func (hub *Hub) Run() {
	for {
		select {
		case client := <-hub.register:
			HandleUserJoinEvent(hub, client)

		case client := <-hub.unregister:
			HandleUserDisconnectEvent(hub, client)

		case payload := <-hub.broadcast:
			BroadcastSocketEventToAllClient(hub, payload)

		case payload := <-hub.specificEmit:
			EmitToSpecificClient(hub, payload)
		}
	}
}
