package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/url"
	"os"
	"path"
	"strings"
	"syscall/js"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/knusbaum/ploker"
)

//go:embed templates/*.tmpl
var templates embed.FS

// fatal rewrites the entire page with an error message and bails out of the
// client. This is yucky. We should do this better.
func fatal(doc js.Value, f string, args ...interface{}) {
	doc.Call("open")
	doc.Call("write", fmt.Sprintf(f, args...))
	doc.Call("close")
	os.Exit(1)
}

// setContent looks for an element (usually a div) with id="content" and replaces its
// HTML with c.
func setContent(doc js.Value, c string) error {
	cdiv := doc.Call("getElementById", "content")
	if !cdiv.Truthy() {
		return errors.New("Failed to get content div")
	}
	cdiv.Set("innerHTML", c) // TODO: error check?
	return nil
}

// appendContent looks for an element (usually a div) with id="content" and appends c
// to its HTML.
func appendContent(doc js.Value, c string) error {
	cdiv := doc.Call("getElementById", "content")
	if !cdiv.Truthy() {
		return errors.New("Failed to get content div")
	}
	v := cdiv.Get("innerHTML")
	cdiv.Set("innerHTML", v.String()+c) // TODO: error check?
	return nil
}

func main() {
	doc := js.Global().Get("document")
	if !doc.Truthy() {
		log.Fatalf("Failed to get document object.")
	}
	win := js.Global().Get("window")
	if !win.Truthy() {
		fatal(doc, "Failed to get window object.")
	}

	href := doc.Get("location").Get("href")
	if !href.Truthy() || href.Type() != js.TypeString {
		fatal(doc, "Failed to get current href.")
	}
	hrefs := href.String()
	u, err := url.Parse(hrefs)
	if err != nil {
		fatal(doc, "Failed to parse current href: %v", err)
	}

	var content string
	var name string
	tmpl, err := template.ParseFS(templates, "templates/*.tmpl")
	if err != nil {
		//content = fmt.Sprintf(
		fatal(doc, "Error parsing templates: %v", err)
	}
	_, id := path.Split(u.Path)

	{

		namechan := make(chan string, 0)
		js.Global().Set("startploker", js.FuncOf(func(this js.Value, args []js.Value) any {

			cdiv := doc.Call("getElementById", "name")
			if !cdiv.Truthy() {
				return errors.New("Failed to get name input")
			}
			namev := cdiv.Get("value") // TODO: error check?
			namechan <- namev.String()
			return nil
		}))

		// Get the user's name
		var sb strings.Builder
		err = tmpl.ExecuteTemplate(&sb, "getname.tmpl", nil)
		if err != nil {
			content = fmt.Sprintf("Error executing template: %v<br/>", err)
		} else {
			content = sb.String()
		}
		setContent(doc, content)
		name = <-namechan
	}

	setContent(doc, "") //TODO: this is critical for some reason.
	dosocket(doc, u.Host, tmpl, id, name)
}

// State is our own version of the state. We deserialize the fields we're interested in.
// This is JS-serialization-compatible with ploker.State.
type State struct {
	// client id -> client state
	Clients map[uint32]struct {
		Name   string
		Bid    int
		DidBid bool
	}
	World struct {
		Reveal    bool
		Countdown int
	}
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

// DoSocket runs the main websocket loop for an active session.
// TODO: Should we retry the connection if we get disconnected?
func dosocket(doc js.Value, hostPort string, tmpl *template.Template, id, name string) {
	start := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, _, err := websocket.Dial(ctx, fmt.Sprintf("ws://%s/sock?id=%s&name=%s", hostPort, url.QueryEscape(id), url.QueryEscape(name)), nil)
	if err != nil {
		// TODO: template
		setContent(doc, fmt.Sprintf("<H1>Session closed: %v</h1>", err))
		return
	}
	defer c.CloseNow()

	{
		// read our client id sent by the server.
		// TODO: It looks like we don't actually need this since we don't need to
		// differentiate our own state as of now. We should consider refactoring.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		var cid uint32
		err := wsjson.Read(ctx, c, &cid)
		if err != nil {
			// TODO: template
			setContent(doc, fmt.Sprintf("<H1>Session closed: %v</h1>", err))
			return
		}
	}

	// These js.Global() functions are callable from javascript. Used in
	// the HTML onclick events to trigger updates to be sent to the server.
	js.Global().Set("bid", js.FuncOf(func(this js.Value, args []js.Value) any {
		if args[0].Type() != js.TypeNumber {
			log.Printf("BID SHOULD BE A NUMBER BUT IS %v", args[0].String())
			return nil
		}
		err := ploker.UpdateUser(ctx, c, "Bid", args[0].Int())
		if err != nil {
			log.Printf("FAILED TO UPDATE BID: %v", args[0].String(), err)
			return nil
		}
		err = ploker.UpdateUser(ctx, c, "DidBid", true)
		if err != nil {
			log.Printf("FAILED TO UPDATE BID: %v", args[0].String(), err)
		}
		return nil
	}))
	js.Global().Set("reveal", js.FuncOf(func(this js.Value, args []js.Value) any {
		ploker.Reveal(ctx, c, "Reveal", true)
		return nil
	}))
	js.Global().Set("reset", js.FuncOf(func(this js.Value, args []js.Value) any {
		ploker.Reset(ctx, c)
		return nil
	}))

	for {
		// Receive updates and re-render the html.
		st, err := ReceiveState(ctx, c)
		if err != nil {
			appendContent(doc, fmt.Sprintf("Failed to get state: %v (%v)", err, time.Now().Sub(start)))
			return
		}

		var sb strings.Builder
		var content string
		err = tmpl.ExecuteTemplate(&sb, "session.tmpl", st)
		if err != nil {
			content += fmt.Sprintf("Error executing template: %v<br/>", err)
		} else {
			content = sb.String()
		}
		setContent(doc, content)
	}
}
