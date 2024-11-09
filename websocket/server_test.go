package websocket

/*
TODO -
1. Create test to start new websocket server
2. Accept new connections:
    a. ensure that when isClosing returns true, the connection isn't accepted
	b. connection manager map is updated
3. Test that when new messages are sent from ingester and filter manager, they are sent to the client
4. Test that when new messages are received from the client, they appear in the ingester channel
5. Test what happens when the connectionManager recv channel is closed unexpectedely, if things are cleaned up
6. Test what happens when a connectionManger recv channel receives a message with the incorrect connection id, that its logged and handled properly
7. Test that a client disconnection is handled properly and is removed from the connection manager map

This may need to be multiple tests. Maybe we start with testing connectionManager and then the server
*/
