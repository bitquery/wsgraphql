package main

import (
	"bytes"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/bitquery/wsgraphql/v1"
	"github.com/bitquery/wsgraphql/v1/compat/gorillaws"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
)

//go:embed playground.html
var playgroundFile []byte

func main() {
	var addr string

	flag.StringVar(&addr, "addr", ":8080", "Address to listen on")
	flag.Parse()

	var foo int

	fooupdates := make(chan int, 1)

	var subscriberID uint64

	type subscriber struct {
		subscription chan interface{}
		ctx          context.Context
		id           uint64
	}

	subscribers := make(map[uint64]*subscriber)
	subscriberadd := make(chan *subscriber, 1)
	subscriberrem := make(chan uint64, 1)

	go func() {
		for {
			select {
			case upd := <-fooupdates:
				foo = upd

				fmt.Println("broadcasting update, new value:", upd)

				for _, sub := range subscribers {
					select {
					case sub.subscription <- upd:
					case <-sub.ctx.Done():
					}
				}
			case add := <-subscriberadd:
				subscribers[add.id] = add

				fmt.Println("added subscriber", add.id)
			case rem := <-subscriberrem:
				close(subscribers[rem].subscription)

				delete(subscribers, rem)

				fmt.Println("removed subscriber", rem)
			}
		}
	}()

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "QueryRoot",
			Fields: graphql.Fields{
				"getFoo": &graphql.Field{
					Description: "Returns most recent foo value",
					Type:        graphql.NewNonNull(graphql.Int),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return foo, nil
					},
				},
			},
		}),
		Mutation: graphql.NewObject(graphql.ObjectConfig{
			Name: "MutationRoot",
			Fields: graphql.Fields{
				"setFoo": &graphql.Field{
					Args: graphql.FieldConfigArgument{
						"value": &graphql.ArgumentConfig{
							Type: graphql.Int,
						},
					},
					Description: "Updates foo value; generating an update to subscribers of fooUpdates",
					Type:        graphql.NewNonNull(graphql.Boolean),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						v, ok := p.Args["value"].(int)
						if ok {
							select {
							case <-p.Context.Done():
								return nil, p.Context.Err()
							case fooupdates <- v:
							}
						}

						return ok, nil
					},
				},
			},
		}),
		Subscription: graphql.NewObject(graphql.ObjectConfig{
			Name: "SubscriptionRoot",
			Fields: graphql.Fields{
				"fooUpdates": &graphql.Field{
					Description: "Updates generated by setFoo mutation",
					Type:        graphql.NewNonNull(graphql.Int),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						// values sent on channel, that were returned from `Subscribe`, will be available here as
						// `p.Source`
						return p.Source, nil
					},
					Subscribe: func(p graphql.ResolveParams) (interface{}, error) {
						// per graphql-go contract, channel returned from `Subscribe` function must have
						// interface{} values
						ch := make(chan interface{}, 1)
						id := atomic.AddUint64(&subscriberID, 1)

						subscriberadd <- &subscriber{
							id:           id,
							subscription: ch,
							ctx:          p.Context,
						}

						go func() {
							<-p.Context.Done()

							subscriberrem <- id
						}()

						return ch, nil
					},
				},
			},
		}),
	})
	if err != nil {
		panic(err)
	}

	srv, err := wsgraphql.NewServer(
		schema,
		wsgraphql.WithKeepalive(time.Second*30),
		wsgraphql.WithConnectTimeout(time.Second*30),
		wsgraphql.WithUpgrader(gorillaws.Wrap(&websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			Subprotocols: []string{
				wsgraphql.WebsocketSubprotocolGraphqlWS.String(),
				wsgraphql.WebsocketSubprotocolGraphqlTransportWS.String(),
			},
		})),
	)
	if err != nil {
		panic(err)
	}

	http.Handle("/query", srv)
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeContent(writer, request, "playground.html", time.Time{}, bytes.NewReader(playgroundFile))
	})

	err = http.ListenAndServe(addr, nil)
	if err != nil {
		panic(err)
	}
}
