package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/TheRebelOfBabylon/tandem/nostr"
	"github.com/gorilla/websocket"
	keys "github.com/nbd-wtf/go-nostr"
)

var (
	relayUrl  = "ws://localhost:5150/"
	stndrdMsg = "This is a test message"
)

func createEvent(pubkey, msg string) nostr.Event {
	return nostr.Event{
		Pubkey:    pubkey,
		CreatedAt: time.Now(),
		Kind:      1,
		Content:   msg,
	}
}

func createClient(url string) (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	return conn, err
}

type Ready struct {
	State uint8
	sync.RWMutex
}

func (r *Ready) SetReady() {
	r.Lock()
	defer r.Unlock()
	r.State = 1
}

func (r *Ready) ReadState() uint8 {
	r.RLock()
	defer r.RUnlock()
	return r.State
}

func main() {
	var (
		wg    sync.WaitGroup
		times []string
		ready Ready
	)
	for i := 0; i < 20000; i++ {
		wg.Add(1)
		go func(i int, r *Ready) {
			defer wg.Done()
			conn, err := createClient(relayUrl)
			if err != nil {
				log.Printf("error connecting to relay for client %v: %v", i, err)
				return
			}
			defer conn.Close()
			sk := keys.GeneratePrivateKey()
			pubKey, err := keys.GetPublicKey(sk)
			if err != nil {
				log.Printf("error creating keys for client %v: %v", i, err)
				return
			}
			event := createEvent(pubKey, stndrdMsg)
			event.EventId = hex.EncodeToString(event.CreateEventId())
			event.SignEvent(sk)
			for {
				if r.ReadState() == 1 {
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
			ts1 := time.Now()
			err = conn.WriteMessage(websocket.TextMessage, event.SerializeEvent())
			times = append(times, fmt.Sprintf("Client %v: %v us", i, time.Since(ts1).Microseconds()))
			if err != nil {
				log.Printf("error writing to relay for client %v: %v", i, err)
				return
			}
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Printf("error reading from relay for client %v: %v", i, err)
				return
			}
			log.Printf("Response from relay for client %v: %s", i, msg)
		}(i, &ready)
		if i%1000 == 0 {
			time.Sleep(3 * time.Second)
		}
	}
	ready.SetReady()
	wg.Wait()
	log.Println("Final results:")
	for _, t := range times {
		fmt.Println(t)
	}

}
