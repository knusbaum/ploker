package ploker

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func GetSessionID(surl string) (string, error) {
	u, err := url.Parse(surl)
	if err != nil {
		//fatal(doc, "Failed to parse current href: %v", err)
		return "", err
	}
	dir, id := path.Split(u.Path)
	if !(strings.HasSuffix(dir, "session") || strings.HasSuffix(dir, "session/")) {
		return "", fmt.Errorf("Not a session URL: %v", surl)
	}
	if id == "" {
		return "", fmt.Errorf("Received empty session id: %v", surl)
	}
	return id, nil
}

const (
	WorldUpdate = iota
	UserUpdate
	ResetUpdate
	RevealUpdate
)

type Update struct {
	Type  int
	Key   string
	Value interface{}
}

// UpdateWorld is used by clients. It sends an update to the server, setting k = v in the
// world state.
func UpdateWorld(ctx context.Context, c *websocket.Conn, k string, v interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return wsjson.Write(ctx, c, &Update{Type: WorldUpdate, Key: k, Value: v})
}

// Reset is used by clients. It sends a reset update to the server, causing the server
// to reset all state for the session.
func Reset(ctx context.Context, c *websocket.Conn) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return wsjson.Write(ctx, c, &Update{Type: ResetUpdate})
}

// UpdateUser is used by clients. It sends an update over connection c, telling the server
// to update the client state associated with its websocket, setting k = v.
func UpdateUser(ctx context.Context, c *websocket.Conn, k string, v interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return wsjson.Write(ctx, c, &Update{Type: UserUpdate, Key: k, Value: v})
}

// Reveal is used by clients. It sends a "reveal" update to the server, causing the server
// to count down visually for all clients before setting k = v in the world state.
func Reveal(ctx context.Context, c *websocket.Conn, k string, v interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return wsjson.Write(ctx, c, &Update{Type: RevealUpdate, Key: k, Value: v})
}

// ReceiveUpdate is used by a server to receive Update objects from clients.
// It returns successfully deserialized Update objects or reports errors for
// deserialization or connection errors.
func ReceiveUpdate(ctx context.Context, c *websocket.Conn) (*Update, error) {
	var u Update
	err := wsjson.Read(ctx, c, &u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// State keeps track of the world state. Clients map client ID to client state.
// World keeps track of global state for a session.
type State struct {
	// client id -> client state
	Clients map[uint32]map[string]interface{}
	World   map[string]interface{}
}

func NewState() State {
	return State{
		Clients: make(map[uint32]map[string]interface{}),
		World:   make(map[string]interface{}),
	}
}

// SendState serializes and sends a State object over connection c.
func SendState(ctx context.Context, c *websocket.Conn, s State) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return wsjson.Write(ctx, c, s)
}

// ReceiveState receives a State object over connection c.
func ReceiveState(ctx context.Context, c *websocket.Conn) (*State, error) {
	var s State
	err := wsjson.Read(ctx, c, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
