package wsgraphql

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bitquery/wsgraphql/v1/apollows"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
	"github.com/stretchr/testify/assert"
)

func testNewServerWebsocketGWS(t *testing.T, srv *httptest.Server) {
	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "1",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `query { getFoo }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "1", msg.ID)
	assert.Equal(t, apollows.OperationData, msg.Type)

	pd, err := msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, 123, pd.Data["getFoo"])

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "1", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "2",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `mutation { setFoo(value: 3) }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "2", msg.ID)
	assert.Equal(t, apollows.OperationData, msg.Type)

	pd, err = msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, map[string]interface{}{
		"setFoo": true,
	}, pd.Data)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "2", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "3",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `mutation { setFoo }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "3", msg.ID)
	assert.Equal(t, apollows.OperationData, msg.Type)

	pd, err = msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, map[string]interface{}{
		"setFoo": false,
	}, pd.Data)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "3", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "4",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `mutation { bar }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "4", msg.ID)
	assert.Equal(t, apollows.OperationError, msg.Type)

	pde, err := msg.Payload.ReadPayloadError()

	assert.NoError(t, err)
	assert.Contains(t, pde.Message, `Cannot query field "bar"`)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "4", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "5",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `subscription { forever }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.WriteJSON(apollows.Message{
		ID:   "5",
		Type: apollows.OperationStop,
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "5", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "6",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `subscription { fooUpdates }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "6", msg.ID)
	assert.Equal(t, apollows.OperationData, msg.Type)

	pd, err = msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, 1, pd.Data["fooUpdates"])

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "6", msg.ID)
	assert.Equal(t, apollows.OperationData, msg.Type)

	pd, err = msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, 2, pd.Data["fooUpdates"])

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "6", msg.ID)
	assert.Equal(t, apollows.OperationData, msg.Type)

	pd, err = msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, 3, pd.Data["fooUpdates"])

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "6", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)

	assert.NoError(t, conn.Close())
}

func testNewServerWebsocketGWTS(t *testing.T, srv *httptest.Server) {
	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlTransportWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "1",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `query { getFoo }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "1", msg.ID)
	assert.Equal(t, apollows.OperationNext, msg.Type)

	pd, err := msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, 123, pd.Data["getFoo"])

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "1", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "2",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `mutation { setFoo(value: 3) }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "2", msg.ID)
	assert.Equal(t, apollows.OperationNext, msg.Type)

	pd, err = msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, map[string]interface{}{
		"setFoo": true,
	}, pd.Data)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "2", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "3",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `mutation { setFoo }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "3", msg.ID)
	assert.Equal(t, apollows.OperationNext, msg.Type)

	pd, err = msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, map[string]interface{}{
		"setFoo": false,
	}, pd.Data)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "3", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "4",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `mutation { bar }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "4", msg.ID)
	assert.Equal(t, apollows.OperationError, msg.Type)

	pde, err := msg.Payload.ReadPayloadErrors()

	assert.NoError(t, err)
	assert.Greater(t, len(pde), 0)
	assert.Contains(t, pde[0].Message, `Cannot query field "bar"`)

	err = conn.WriteJSON(apollows.Message{
		ID:   "5",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `subscription { forever }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.WriteJSON(apollows.Message{
		ID:   "5",
		Type: apollows.OperationStop,
	})

	assert.NoError(t, err)

	err = conn.WriteJSON(apollows.Message{
		ID:   "6",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `subscription { fooUpdates }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "6", msg.ID)
	assert.Equal(t, apollows.OperationNext, msg.Type)

	pd, err = msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, 1, pd.Data["fooUpdates"])

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "6", msg.ID)
	assert.Equal(t, apollows.OperationNext, msg.Type)

	pd, err = msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, 2, pd.Data["fooUpdates"])

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "6", msg.ID)
	assert.Equal(t, apollows.OperationNext, msg.Type)

	pd, err = msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, 3, pd.Data["fooUpdates"])

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "6", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)

	assert.NoError(t, conn.Close())
}

func TestNewServerWebsocketGWS(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlWS, WithConnectTimeout(time.Second))

	defer srv.Close()

	testNewServerWebsocketGWS(t, srv)
}

func TestNewServerWebsocketGWTS(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlTransportWS, WithConnectTimeout(time.Second))

	defer srv.Close()

	testNewServerWebsocketGWTS(t, srv)
}

func TestNewServerWebsocketGWSGWTS(t *testing.T) {
	srv := testNewServerProtocols(
		t,
		[]apollows.Protocol{apollows.WebsocketSubprotocolGraphqlWS, apollows.WebsocketSubprotocolGraphqlTransportWS},
		WithProtocol(apollows.WebsocketSubprotocolGraphqlTransportWS),
		WithConnectTimeout(time.Second),
	)

	defer srv.Close()

	testNewServerWebsocketGWS(t, srv)
	testNewServerWebsocketGWTS(t, srv)
}

func TestNewServerWebsocketProtocolMismatch(t *testing.T) {
	srv := testNewServerProtocols(
		t,
		[]apollows.Protocol{apollows.WebsocketSubprotocolGraphqlWS, apollows.WebsocketSubprotocolGraphqlTransportWS},
		WithProtocol(apollows.WebsocketSubprotocolGraphqlTransportWS),
		WithConnectTimeout(time.Second),
	)

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{"foo"},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.ErrorContains(t, err, apollows.ErrUnknownProtocol.Error())
}

func TestNewServerWebsocketKeepalive(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlWS, WithKeepalive(time.Millisecond*10))

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationKeepAlive, msg.Type)
}

func TestNewServerWebsocketTerminateGWS(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlWS)

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "",
		Type: apollows.OperationTerminate,
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.ErrorContains(t, err, "requested")
}

func TestNewServerWebsocketTerminateGTWS(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlTransportWS)

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlTransportWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "",
		Type: apollows.OperationTerminate,
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.ErrorContains(t, err, "Unauthorized")
}

func TestNewServerWebsocketTimeoutGWS(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlWS, WithConnectTimeout(1))

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	time.Sleep(time.Millisecond * 10)

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionError, msg.Type)
}

func TestNewServerWebsocketTimeoutGTWS(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlTransportWS, WithConnectTimeout(1))

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlTransportWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	time.Sleep(time.Millisecond * 10)

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.ErrorContains(t, err, "4408: Connection initialisation timeout")
}

func TestNewServerWebsocketReinitGWS(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlWS)

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionError, msg.Type)

	pde, err := msg.Payload.ReadPayloadError()
	assert.NoError(t, err)

	assert.Contains(t, pde.Message, "Too many initialisation requests")
}

func TestNewServerWebsocketReinitGTWS(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlTransportWS)

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlTransportWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.ErrorContains(t, err, "4429: Too many initialisation requests")
}

func TestNewServerWebsocketOperationRestartGWS(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlWS, WithConnectTimeout(time.Second))

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "1",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `subscription { forever }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.WriteJSON(apollows.Message{
		ID:   "1",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `subscription { forever }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionError, msg.Type)

	pde, err := msg.Payload.ReadPayloadError()

	assert.NoError(t, err)
	assert.Contains(t, pde.Message, "Subscriber for 1 already exists")
}

func TestNewServerWebsocketOperationRestartGTWS(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlTransportWS, WithConnectTimeout(time.Second))

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlTransportWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "1",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `subscription { forever }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.WriteJSON(apollows.Message{
		ID:   "1",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `subscription { forever }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.ErrorContains(t, err, "4409: Subscriber for 1 already exists")
}

func TestNewServerWebsocketOperationInvalidGWS(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlWS, WithConnectTimeout(time.Second))

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "1",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: "foo",
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "1", msg.ID)
	assert.Equal(t, apollows.OperationError, msg.Type)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "1", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)
}

func TestNewServerWebsocketOperationInvalidGTWS(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlTransportWS, WithConnectTimeout(time.Second))

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlTransportWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "1",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: "foo",
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.ErrorContains(t, err, "4400: Invalid message")
}

func TestNewServerWebsocketOperationErrorGWS(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlWS, WithConnectTimeout(time.Second))

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "1",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `query { getError }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "1", msg.ID)
	assert.Equal(t, apollows.OperationData, msg.Type)

	pd, err := msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 1)
	assert.Contains(t, pd.Errors[0].Message, "someerr")

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "1", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)
}

func TestNewServerWebsocketOperationErrorGTWS(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlTransportWS, WithConnectTimeout(time.Second))

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlTransportWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "1",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `query { getError }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "1", msg.ID)
	assert.Equal(t, apollows.OperationNext, msg.Type)

	pd, err := msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 1)
	assert.Contains(t, pd.Errors[0].Message, "someerr")

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "1", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)
}

func TestNewServerWebsocketPingGTWS(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlTransportWS, WithConnectTimeout(time.Second))

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlTransportWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		Type: apollows.OperationPing,
		Payload: apollows.Data{
			Value: map[string]interface{}{
				"foo": 123,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationPong, msg.Type)

	var m map[string]interface{}

	err = json.Unmarshal(msg.Payload.RawMessage, &m)

	assert.NoError(t, err)
	assert.EqualValues(t, 123, m["foo"])
}

func TestNewServerWebsocketCombineErrorsGWS(t *testing.T) {
	ex1 := &testExt{}
	ex2 := &testExt{}

	var opts []ServerOption

	opts = append(opts, WithUpgrader(testWrapper{
		Upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			Subprotocols:    []string{apollows.WebsocketSubprotocolGraphqlWS.String()},
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}), WithConnectTimeout(time.Second))

	opts = append(opts, WithProtocol(apollows.WebsocketSubprotocolGraphqlWS))

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:       "QueryRoot",
			Interfaces: nil,
			Fields: graphql.Fields{
				"getFoo": &graphql.Field{
					Type: graphql.Int,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return 123, nil
					},
				},
				"getError": &graphql.Field{
					Type: graphql.Int,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return nil, errors.New("someerr")
					},
				},
			},
		}),
		Extensions: []graphql.Extension{
			ex1,
			ex2,
		},
	})

	assert.NoError(t, err)

	server, err := NewServer(schema, opts...)

	assert.NoError(t, err)
	assert.NotNil(t, server)

	srv := httptest.NewServer(server)

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{apollows.WebsocketSubprotocolGraphqlWS.String()},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "1",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `query { getError }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "1", msg.ID)
	assert.Equal(t, apollows.OperationError, msg.Type)

	pd, err := msg.Payload.ReadPayloadError()

	assert.NoError(t, err)
	assert.NotNil(t, pd.Extensions["errors"])
	assert.Len(t, pd.Extensions["errors"], 2)
}
