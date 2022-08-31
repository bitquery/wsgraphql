// Package wsgraphql provides interfaces for server and client
package wsgraphql

import (
	"github.com/bitquery/wsgraphql/v1/apollows"
)

// WebsocketSubprotocolGraphqlWS websocket subprotocol expected by subscriptions-transport-ws implementations
const WebsocketSubprotocolGraphqlWS = apollows.WebsocketSubprotocolGraphqlWS

// WebsocketSubprotocolGraphqlTransportWS websocket subprotocol expected by graphql-ws implementations
const WebsocketSubprotocolGraphqlTransportWS = apollows.WebsocketSubprotocolGraphqlTransportWS
