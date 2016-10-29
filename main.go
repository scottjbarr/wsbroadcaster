package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

var (
	rr redisReceiver
	rw redisWriter
)

func main() {
	redisURL := os.Getenv("REDIS_URL")
	redisPool, err := newRedisPool(redisURL)
	if err != nil {
		log.Fatalf("Unable to create Redis pool : %v", redisURL)
	}

	redisKey := os.Getenv("REDIS_KEY")
	if redisKey == "" {
		log.Fatalf("REDIS_KEY must be supplied")
	}

	rr = newRedisReceiver(redisPool, redisKey)
	go rr.run()

	rw = newRedisWriter(redisPool, redisKey)
	go rw.run()

	bind := os.Getenv("BIND")
	if bind == "" {
		log.Printf("$BIND must be set")
		os.Exit(1)
	}

	r := mux.NewRouter()
	r.HandleFunc("/ws", handleWebsocket)

	http.ListenAndServe(bind, r)
}
