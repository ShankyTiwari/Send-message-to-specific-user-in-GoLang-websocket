const domElement = document.querySelector(".chat__app-container");

class App extends React.Component {
    constructor() {
        super();
        this.state = {
            chatUserList: [],
            message: null,
            selectedUserID: null,
        }
        this.webSocketConnection = null;
    }

    componentDidMount() {
        this.setWebSocketConnection();
        this.subscribeToSocketMessage();
    }

    setWebSocketConnection() {
        const username = prompt("What's Your name");
        if (window["WebSocket"]) {
            const socketConnection = new WebSocket("ws://" + document.location.host + "/ws/" + username);
            this.webSocketConnection = socketConnection;
        }
    }

    subscribeToSocketMessage = () => {
        if (this.webSocketConnection === null) {
            return;
        }

        this.webSocketConnection.onclose = (evt) => {
            this.setState({
                message: 'Your Connection is closed.',
                chatUserList: []
            });
        };

        this.webSocketConnection.onmessage = (event) => {
            try {
                const socketPayload = JSON.parse(event.data);
                switch (socketPayload.eventName) {
                    case 'join':
                        if (!socketPayload.eventPayload) {
                            return
                        }
                        this.setState({
                            chatUserList: socketPayload.eventPayload
                        });

                        break;

                    case 'disconnect':
                         if (!socketPayload.eventPayload) {
                            return
                         }
                         this.setState({
                            chatUserList: socketPayload.eventPayload
                         });

                        break;

                    case 'message response':

                        if (!socketPayload.eventPayload) {
                            return
                        }

                        const messageContent = socketPayload.eventPayload;
                        const sentBy = messageContent.username ? messageContent.username : 'An unnamed fellow'
                        const actualMessage = messageContent.message;
                        
                        this.setState({
                            message: `${sentBy} says: ${actualMessage}`
                        });

                        break;

                    default:
                        break;
                }
            } catch (error) {
                console.log(error)
                console.warn('Something went wrong while decoding the Message Payload')
            }
        };
    }

    handleKeyPress = (event) => {
        try {
            if (event.key === 'Enter') {
                if (!this.webSocketConnection) {
                    return false;
                }
                if (!event.target.value) {
                    return false;
                }

                this.webSocketConnection.send(JSON.stringify({
                    EventName: 'message',
                    EventPayload: {
                        socketID: this.state.selectedUserID,
                        message: event.target.value
                    },
                }));
            
                event.target.value = '';
            }            
        } catch (error) {
            console.log(error)
            console.warn('Something went wrong while decoding the Message Payload')
        }
    }

    setNewUserToChat = (event) => {
        if (event.target && event.target.value) {
            if (event.target.value === "select-user") {
                alert("Select a user to chat");
                return;
            }
            this.setState({
                selectedUserID: event.target.value
            })   
        }
    }
    
    getChatList() {
        if (this.state.chatUserList.length === 0) {
            return(
                <h3>No one has joined yet</h3>
            )
        }
        return (
            <div className="chat__list-container">
                <p>Select a user to chat</p>
                <select onChange={this.setNewUserToChat}>
                    <option value={'select-user'} className="username-list">Select User</option>
                    {
                        this.state.chatUserList.map(user => {
                            return (
                                <option value={user.socketID} className="username-list">
                                    {user.username}
                                </option>
                            )
                        })
                    }
                </select>
            </div>
        );
    }

    getChatContainer() {
        return (
            <div class="chat__message-container">
                <div class="message-container">
                    {this.state.message}
                </div>
                <input type="text" id="message-text" size="64" autofocus placeholder="Type Your message" onKeyPress={this.handleKeyPress}/>
            </div>
        );
    }

    render() {
        return (
            <React.Fragment>
                {this.getChatList()}
                {this.getChatContainer()}
            </React.Fragment>
        );
    }
}

ReactDOM.render(<App />, domElement)