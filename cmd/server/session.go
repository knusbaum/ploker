package main

import (
	"context"
	"fmt"
	"log"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/knusbaum/ploker"
)

// sessionMgr keeps track of all sessions on the server.
// It is synchronized and drops empty sessions when all clients
// leave.
type sessionMgr struct {
	Sessions map[string]*session
	m        sync.Mutex
}

// NewSessionMgr constructs a sessionMgr
func NewSessionMgr() *sessionMgr {
	return &sessionMgr{
		Sessions: make(map[string]*session),
	}
}

// getSession will create a new session, or retrieve the existing session for
// a given session ID.
func (m *sessionMgr) getSession(id string) *session {
	m.m.Lock()
	defer m.m.Unlock()
	if s, ok := m.Sessions[id]; ok {
		return s
	}
	s := &session{
		State:       ploker.NewState(),
		clients:     make(map[*websocket.Conn]uint32),
		lastContact: time.Now(),
	}
	m.Sessions[id] = s
	return s
}

// drop will drop a given client from the session with session ID id.
// If the session becomes empty, it will remove the session and all its data
// from the session manager.
func (m *sessionMgr) drop(id string, c *websocket.Conn) {
	m.m.Lock()
	defer m.m.Unlock()
	s, ok := m.Sessions[id]
	if !ok {
		return
	}
	if s.drop(c) == 0 {
		// This session is now empty
		delete(m.Sessions, id)
	}
}

// A session represents the shared state of a group of clients.
type session struct {
	// State contains all of the state shared among the clients.
	// This is the data that is broadcast to the clients and the State
	// the clients update with Update messages.
	State ploker.State

	// clients maps client conn -> client uid. This is used to manage
	// client State data when websocket events happen, i.e. updates or
	// connection drops.
	clients map[*websocket.Conn]uint32

	// lastContact keeps track of the last requested client update.
	lastContact time.Time
	m           sync.Mutex
}

// UpdateWorld updates the world state, setting k = v
func (s *session) UpdateWorld(k string, v interface{}) {
	s.m.Lock()
	defer s.m.Unlock()
	s.lastContact = time.Now()
	s.State.World[k] = v
}

// Update a user's state, setting key k = value v for user with id uid.
func (s *session) UpdateUser(uid uint32, k string, v interface{}) error {
	s.m.Lock()
	defer s.m.Unlock()
	s.lastContact = time.Now()
	client, ok := s.State.Clients[uid]
	if !ok {
		return fmt.Errorf("No client for id %v", uid)
	}
	client[k] = v
	return nil
}

// Broadcast sends the state of the world to all connected clients.
func (s *session) broadcast(ctx context.Context) {
	s.m.Lock()
	defer s.m.Unlock()
	for c := range s.clients {
		err := ploker.SendState(ctx, c, s.State)
		if err != nil {
			// I don't think we should drop the client here since the
			// main client loop should handle that when the conn drops.
			log.Printf("Failed to send state to client: %v", err)
		}
	}
}

// LastContact reports the last time any client in a session sent an update message.
// This is useful for timing out sessions.
func (s *session) LastContact() time.Time {
	s.m.Lock()
	defer s.m.Unlock()
	return s.lastContact
}

// Reset will clear a session, keeping the clients and their names, but resetting
// all other client and world data.
func (s *session) reset() {
	s.m.Lock()
	defer s.m.Unlock()
	s.lastContact = time.Now()
	s.State.World = make(map[string]interface{})
	for i := range s.State.Clients {
		// TODO: gross special case. We want to retain the
		// client's name when resetting the state of the session.
		name := s.State.Clients[i]["name"]
		s.State.Clients[i] = make(map[string]interface{})
		s.State.Clients[i]["name"] = name
	}
}

// Drop drops the client data associated with a websocket connection c.
func (s *session) drop(c *websocket.Conn) int {
	s.m.Lock()
	defer s.m.Unlock()
	//	return s.unsafe_drop(c)
	//}

	//func (s *session) unsafe_drop(c *websocket.Conn) int {
	cid := s.clients[c]
	delete(s.State.Clients, cid)
	delete(s.clients, c)
	return len(s.clients)
}

// NewClient adds a new client named name to the session and returns the
// client's numeric uid, by which the client is referenced in the session.
func (s *session) NewClient(c *websocket.Conn, name string) uint32 {
	s.m.Lock()
	defer s.m.Unlock()

	cid := mrand.Uint32()
	s.clients[c] = cid
	s.State.Clients[cid] = map[string]interface{}{
		"name": name,
	}
	return cid
}
