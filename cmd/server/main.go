package main

import (
	"context"
	crand "crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/knusbaum/ploker"
)

// randID generates a random ID for a new session.
func randID() (string, error) {
	// 64-bit random is big enough to prevent brute force searching
	// for active sessions.
	var ba [8]byte
	bs := ba[:]
	// This is crypto/rand to prevent attackers from guessing
	// new sessions. Probably overkill.
	_, err := crand.Read(bs)
	if err != nil {
		return "", err
	}
	// Hex is nice and readable and most importantly URL-safe.
	r := hex.EncodeToString(bs)
	return r, nil
}

//go:embed templates/*.tmpl
var templates embed.FS

func main() {
	sm := NewSessionMgr()
	tmpl, err := template.ParseFS(templates, "templates/*.tmpl")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v\n", err)
	}
	log.Printf("Have templates: %v", tmpl.DefinedTemplates())

	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		e := json.NewEncoder(w)
		err := e.Encode(sm)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode: %v", err), 500)
		}
	})
	http.HandleFunc("/ploker.wasm", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/wasm")
		http.ServeFile(w, r, "ploker.wasm")
	})
	http.HandleFunc("/wasm_exec.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "wasm_exec.js")
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, "File Not Found", 404)
			return
		}
		err = tmpl.ExecuteTemplate(w, "home.tmpl", nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("Unknown server error: %v", err), 500)
			return
		}
	})
	http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {

	})
	http.HandleFunc("/session/", func(w http.ResponseWriter, r *http.Request) {
		_, err := ploker.GetSessionID(r.URL.Path)
		if err != nil {
			rid, err := randID()
			if err != nil {
				log.Printf("Failed to generate random ID: %v", err)
				http.Error(w, fmt.Sprintf("Unknown server error: %v", err), 500)
				return
			}
			http.Redirect(w, r, fmt.Sprintf("/session/%v", rid), 302)
			return

		}
		err = tmpl.ExecuteTemplate(w, "session.tmpl", nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("Unknown server error: %v", err), 500)
			return
		}
	})

	// This is the main websocket routine. Handles client updates and broadcasting
	// state to clients.
	http.HandleFunc("/sock", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
		defer cancel()
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			// TODO: make configurable
			OriginPatterns: []string{"poker.fritterware.org", "localhost:8080"},
		})
		if err != nil {
			log.Printf("Failed to accept socket connection: %v", err)
			return
		}
		defer c.CloseNow()

		sid := r.URL.Query().Get("id")
		name := r.URL.Query().Get("name")
		if sid == "" {
			log.Printf("expected non-empty session id.")
			return
		}
		if name == "" {
			log.Printf("expected non-empty client name.")
			return
		}
		sess := sm.getSession(sid)
		cid := sess.NewClient(c, name)
		// broadcast after cleanup to propagate client drop
		defer sess.broadcast(ctx)
		defer sm.drop(sid, c)

		{
			// send the client id
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			err = wsjson.Write(ctx, c, &cid)
			if err != nil {
				log.Printf("Failed to establish connection with client.\n")
				return
			}
		}

		go func() {
			defer log.Printf("Stopping pinging for client %p", c)
			for {
				// Keep the websocket alive through any proxy
				// by pinging every 10 seconds. Most proxies seem
				// to kill connections after around a minute.
				// TODO: Configurable?
				time.Sleep(10 * time.Second)
				if ctx.Err() != nil {
					return
				}
				if time.Now().After(sess.LastContact().Add(20 * time.Minute)) {
					return
				}
				err := c.Ping(ctx)
				if err != nil {
					log.Printf("Ping error: %v\n", err)
					return
				}
				log.Printf("ping %v@%v", cid, sid)
			}
		}()

		sess.broadcast(ctx)
		for {
			up, err := ploker.ReceiveUpdate(ctx, c)
			if err != nil {
				log.Printf("Error receiving update from client %v: %v", cid, err)
				return
			}
			switch up.Type {
			case ploker.WorldUpdate:
				sess.UpdateWorld(up.Key, up.Value)
			case ploker.UserUpdate:
				sess.UpdateUser(cid, up.Key, up.Value)
			case ploker.ResetUpdate:
				sess.reset()
			case ploker.RevealUpdate:
				go func() {
					for i := 3; i > 0; i-- {
						sess.UpdateWorld("countdown", i)
						sess.broadcast(ctx)
						time.Sleep(1 * time.Second)
					}
					sess.UpdateWorld("countdown", 0)
					sess.UpdateWorld(up.Key, up.Value)
					sess.broadcast(ctx)
				}()
			default:
				log.Printf("Unknown update type %v", up.Type)
				continue
			}
			sess.broadcast(ctx)
		}
		// TODO: Should there be a normal way of closing the
		// connection gracefully?
		// c.Close(websocket.StatusNormalClosure, "")
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
