// Package pluginkit is a Persona plugin's side of the plugin protocol (Persona's docs/plugins.md「进程插件协议」),
// standing alone: JSON lines on stdin / stdout, initialize, ping, and namespaced methods. It depends on nothing of
// Persona's, so a plugin builds on its own.
package pluginkit

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

// Protocol is the plugin protocol's version.
const Protocol = 1

// Error codes.
const (
	CodeNotInitialized = "not_initialized"
	CodeUnknownMethod  = "unknown_method"
	CodeWrongProtocol  = "wrong_protocol"
	CodeNoContribution = "no_contribution"
	CodeUnsupported    = "unsupported"
)

// Error is an error with a code the hub acts on.
type Error struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func (e *Error) Error() string { return e.Message }

// Errorf is an error with a code.
func Errorf(code, format string, args ...any) error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// InitParams is what initialize carries (the parts a plugin uses).
type InitParams struct {
	Protocol int `json:"protocol"`
	Hub      struct {
		Build           int64          `json:"build"`
		ExtensionPoints map[string]int `json:"extension_points"`
		// Role is the hub's place in its group: "center" or "member" ("" from a hub before roles: a center).
		Role string `json:"role"`
		// Capabilities are what the hub does that a plugin may go by (HubInboxGroup); none from a hub before them.
		Capabilities []string `json:"capabilities"`
	} `json:"hub"`
	Plugin struct {
		ID      string `json:"id"`
		Version string `json:"version"`
		Dir     string `json:"dir"`
		DataDir string `json:"data_dir"`
	} `json:"plugin"`
	Config  json.RawMessage     `json:"config"`
	Secrets map[string]string   `json:"secrets"`
	Granted json.RawMessage     `json:"granted"`
	Active  map[string][]string `json:"active"`
}

// Hub capabilities (InitParams.Hub.Capabilities).
const (
	// HubInboxGroup: the hub relates a connector's batches of new messages to inbox events (Persona's docs/inbox.md
	// 「事件合成」): a connector that declares the capability "group" hands batches and writes the events' meta.
	HubInboxGroup = "inbox_group"
)

// HubCan says whether the hub said it has capability c.
func (p InitParams) HubCan(c string) bool {
	for _, have := range p.Hub.Capabilities {
		if have == c {
			return true
		}
	}
	return false
}

// Handler answers one method ("quota.query").
type Handler func(ctx context.Context, hub *Hub, params json.RawMessage) (any, error)

// Plugin is a plugin program.
type Plugin struct {
	ID, Version string
	// Manifest is its persona-plugin.json, answered to manifest (needed when it runs elsewhere).
	Manifest json.RawMessage
	// Contributes is what initialize says it contributes (a subset of the manifest's).
	Contributes map[string]any
	// Init runs once with initialize's parameters; hub is for the plugin's own calls and events from then on.
	Init     func(p InitParams, hub *Hub) error
	Handlers map[string]Handler
}

type message struct {
	ID     string          `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Event  string          `json:"event,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

// Hub is the hub, for the plugin's calls.
type Hub struct {
	w       io.Writer
	wmu     sync.Mutex
	next    atomic.Int64
	mu      sync.Mutex
	pending map[string]chan message
}

func (h *Hub) write(m message) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	h.wmu.Lock()
	defer h.wmu.Unlock()
	_, err = h.w.Write(append(b, '\n'))
	return err
}

// Notify sends the hub a one-way event ({"event", "params"}): connector.changed, api.broadcast.
func (h *Hub) Notify(event string, params any) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return h.write(message{Event: event, Params: raw})
}

// Call calls a hub method and decodes its result into out.
func (h *Hub) Call(ctx context.Context, method string, params, out any) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	id := fmt.Sprintf("p%d", h.next.Add(1))
	ch := make(chan message, 1)
	h.mu.Lock()
	h.pending[id] = ch
	h.mu.Unlock()
	defer func() { h.mu.Lock(); delete(h.pending, id); h.mu.Unlock() }()
	if err := h.write(message{ID: id, Method: method, Params: raw}); err != nil {
		return err
	}
	select {
	case m := <-ch:
		if m.Error != nil {
			return m.Error
		}
		if out != nil && len(m.Result) > 0 {
			return json.Unmarshal(m.Result, out)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Run serves p on stdin / stdout until the hub closes stdin.
func Run(p Plugin) error { return Serve(context.Background(), os.Stdin, os.Stdout, p) }

// Serve serves p on in / out.
func Serve(ctx context.Context, in io.Reader, out io.Writer, p Plugin) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	hub := &Hub{w: out, pending: map[string]chan message{}}
	var initialized atomic.Bool
	var initOnce sync.Once
	var initErr error
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 64<<10), 64<<20)
	for sc.Scan() {
		var m message
		if json.Unmarshal(sc.Bytes(), &m) != nil {
			continue
		}
		switch {
		case m.Method != "":
			go func(m message) {
				res, err := func() (any, error) {
					switch m.Method {
					case "initialize":
						var ip InitParams
						if err := json.Unmarshal(m.Params, &ip); err != nil {
							return nil, err
						}
						if ip.Protocol != Protocol {
							return nil, Errorf(CodeWrongProtocol, "protocol %d, not %d", ip.Protocol, Protocol)
						}
						initOnce.Do(func() {
							if p.Init != nil {
								initErr = p.Init(ip, hub)
							}
							if initErr == nil {
								initialized.Store(true)
							}
						})
						if initErr != nil {
							return nil, initErr
						}
						return map[string]any{"protocol": Protocol, "id": p.ID, "version": p.Version, "contributes": p.Contributes}, nil
					case "manifest":
						if len(p.Manifest) == 0 {
							return nil, Errorf(CodeUnknownMethod, "no manifest here: it is in the package")
						}
						return p.Manifest, nil
					}
					if !initialized.Load() {
						return nil, Errorf(CodeNotInitialized, "not initialized")
					}
					if m.Method == "ping" {
						return struct{}{}, nil
					}
					h := p.Handlers[m.Method]
					if h == nil {
						return nil, Errorf(CodeUnknownMethod, "unknown method %s", m.Method)
					}
					return h(ctx, hub, m.Params)
				}()
				reply := message{ID: m.ID}
				var coded *Error
				switch {
				case errors.As(err, &coded):
					reply.Error = coded
				case err != nil:
					reply.Error = &Error{Message: err.Error()}
				default:
					b, merr := json.Marshal(res)
					if merr != nil {
						reply.Error = &Error{Message: merr.Error()}
					} else {
						reply.Result = b
					}
				}
				_ = hub.write(reply)
			}(m)
		case m.Event != "":
		case m.ID != "":
			hub.mu.Lock()
			ch := hub.pending[m.ID]
			hub.mu.Unlock()
			if ch != nil {
				ch <- m
			}
		}
	}
	return sc.Err()
}

// Contribution is the contribution a call names.
func Contribution(params json.RawMessage) string {
	var t struct {
		Contribution string `json:"contribution"`
	}
	_ = json.Unmarshal(params, &t)
	return strings.TrimSpace(t.Contribution)
}
