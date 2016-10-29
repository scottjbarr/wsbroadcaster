package main

import (
	"io"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// TODO permit an array of allowed origins, from config?
			if r.Header.Get("Origin") == "http://localhost:3000" {
				return true
			}

			return false
		},
	}
)

// handleWebsocket connection
func handleWebsocket(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to websockets : %v", err)
		http.Error(w, "Error upgrading to websockets", 400)
		return
	}

	id := rr.register(ws)

	for {
		mt, data, err := ws.ReadMessage()
		ctx := map[string]interface{}{
			"mt":   mt,
			"data": data,
			"err":  err,
		}

		if err != nil {
			if err == io.EOF {
				log.Printf("Websocket closed : %+v", ctx)
			} else {
				log.Printf("Error reading websocket message : %+v", ctx)
			}

			break
		}
		switch mt {
		case websocket.TextMessage:
			rw.publish(data)
		default:
			log.Printf("Unknown Message! : %+v", ctx)
		}
	}

	rr.deRegister(id)

	ws.WriteMessage(websocket.CloseMessage, []byte{})
}
