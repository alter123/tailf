package main

import (
	"fmt"
	"log"
	"net/http"
)

// Broker to maintain the connection/exchanges with the clients/workers
// Implementation boilerplate is referred from https://github.com/kljensen/golang-html5-sse-example
type Broker struct {
	// Maintain map of all subscribers
	// value is to determine if the subscriber is still active
	subscribers map[chan string]bool

	// channel to add new subscribers
	subscribe chan chan string

	// unsubscribe channel
	// channel `into` which disconnected subscribers should be pushed
	unsubscribe chan (chan string) // channel of subscribers

	// channel to send messages to subscribers
	messages chan string
}

func NewBroker() *Broker {
	b := &Broker{
		subscribers: make(map[chan string]bool),
		subscribe:   make(chan chan string),
		unsubscribe: make(chan chan string),
		messages:    make(chan string),
	}
	return b
}

func (b *Broker) start() {
	go func() {
		for {
			select {
			case s := <-b.subscribe:
				b.subscribers[s] = true
				log.Println("subscriber added")

			case s := <-b.unsubscribe:
				delete(b.subscribers, s)
				close(s)
				log.Println("subscriber removed")

			case m := <-b.messages:
				for s := range b.subscribers {
					s <- m
				}

				log.Println("broadcasted message to subscribers", len(b.subscribers))
			}
		}
	}()
}

func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Make sure that the writer supports flushing.
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	subscriberChan := make(chan string)
	b.subscribe <- subscriberChan

	// Listen to the closing of the http connection via the CloseNotifier
	go func() {
		<-ctx.Done()
		// Remove this client from the map of attached clients
		// when `EventHandler` exits.
		b.unsubscribe <- subscriberChan
		log.Println("HTTP connection just closed.")
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	// Don't close the connection, instead loop endlessly.
	for {
		msg, open := <-subscriberChan
		if !open {
			break
		}

		// Write to the ResponseWriter, `w`.
		fmt.Fprintf(w, "data: Message: %s\n\n", msg)

		// Flush the response.  This is only possible if
		// the response supports streaming.
		f.Flush()
	}

	log.Println("Finished HTTP request at ", r.URL.Path)
}
