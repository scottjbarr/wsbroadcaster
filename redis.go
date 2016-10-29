package main

import (
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/satori/go.uuid"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/websocket"
)

// const (
// 	CHANNEL = "demo:data-in"
// )

func newRedisPool(us string) (*redis.Pool, error) {
	u, err := url.Parse(us)
	if err != nil {
		return nil, err
	}

	var password string
	if u.User != nil {
		password, _ = u.User.Password()
	}

	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", u.Host)
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}, nil
}

// redisReceiver receives messages from Redis and broadcasts them to all
// registered websocket connections that are Registered.
type redisReceiver struct {
	pool       *redis.Pool
	sync.Mutex // Protects the conns map
	conns      map[string]*websocket.Conn
	key        string
}

// newRedisReceiver creates a redisReceiver that will use the provided
// rredis.Pool.
func newRedisReceiver(pool *redis.Pool, key string) redisReceiver {
	return redisReceiver{
		pool:  pool,
		conns: make(map[string]*websocket.Conn),
		key:   key,
	}
}

// run receives pubsub messages from Redis after establishing a connection.
// When a valid message is received it is broadcast to all connected websockets
func (rr *redisReceiver) run() {
	conn := rr.pool.Get()
	defer conn.Close()
	psc := redis.PubSubConn{conn}
	psc.Subscribe(rr.key)

	for {
		switch v := psc.Receive().(type) {
		case redis.Message:
			log.Printf("Redis Message Received : channel=%v message=%v",
				v.Channel,
				string(v.Data))

			// msg, err := validateMessage(v.Data)
			// if err != nil {
			// 	log.Printf("Error unmarshalling message from Redis : err=%v : data=%v : msg=%v", err, v.Data, msg)
			// 	continue
			// }
			rr.broadcast(v.Data)
		case redis.Subscription:
			log.Printf("Redis subscription received : channel=%v : kind=%v : count=%v", v.Channel, v.Kind, v.Count)
		case error:
			log.Printf("Error while subscribed to Redis channel %s : %v",
				rr.key,
				v)
		default:
			log.Println("Unknown Redis receive during subscription : v=%v", v)
		}
	}
}

// broadcast the provided message to all connected websocket connections.
// If an error occurs while writting a message to a websocket connection it is
// closed and deregistered.
func (rr *redisReceiver) broadcast(data []byte) {
	rr.Mutex.Lock()
	defer rr.Mutex.Unlock()
	for id, conn := range rr.conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("Error writting data to connection! Closing and removing Connection : id=%v : data=%v : err=%v : conn=%v", id, data, err, conn)

			rr.deRegister(id)
		}
	}
}

// register the websocket connection with the receiver and return a unique
// identifier for the connection. This identifier can be used to deregister the
// connection later
func (rr *redisReceiver) register(conn *websocket.Conn) string {
	rr.Mutex.Lock()
	defer rr.Mutex.Unlock()
	id := uuid.NewV4().String()
	rr.conns[id] = conn
	return id
}

// deRegister the connection by closing it and removing it from our list.
func (rr *redisReceiver) deRegister(id string) {
	rr.Mutex.Lock()
	defer rr.Mutex.Unlock()
	conn, ok := rr.conns[id]
	if ok {
		conn.Close()
		delete(rr.conns, id)
	}
}

// redisWriter publishes messages to the Redis key
type redisWriter struct {
	pool     *redis.Pool
	messages chan []byte
	key      string
}

func newRedisWriter(pool *redis.Pool, key string) redisWriter {
	return redisWriter{
		pool:     pool,
		messages: make(chan []byte),
		key:      key,
	}
}

// run the main redisWriter loop that publishes incoming messages to Redis.
func (rw *redisWriter) run() {
	conn := rw.pool.Get()
	defer conn.Close()

	for data := range rw.messages {
		ctx := map[string]interface{}{
			"data": data,
		}

		if err := conn.Send("PUBLISH", rw.key, data); err != nil {
			ctx["err"] = err
			log.Printf("Unable to publish message to Redis : %+v", ctx)
		}
		if err := conn.Flush(); err != nil {
			ctx["err"] = err
			log.Printf("Unable to flush published message to Redis : %+v", ctx)
		}
	}
}

// publish to Redis via channel.
func (rw *redisWriter) publish(data []byte) {
	rw.messages <- data
}
