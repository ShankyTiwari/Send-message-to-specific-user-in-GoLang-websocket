package handlers

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
}

// NewHub will will give an instance of an Hub
func NewHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// Run will execute Go Routines to check incoming Socket events
func (hub *Hub) Run() {
	for {
		select {
		case client := <-hub.register:
			HandleUserRegisterEvent(hub, client)

		case client := <-hub.unregister:
			HandleUserDisconnectEvent(hub, client)
		}
	}
}
