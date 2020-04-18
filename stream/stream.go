package stream

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tmpim/juroku"
)

const minBufferSize = 3

// Subscription is subscription.
type Subscription uint32

// Possible subscription flags.
const (
	SubscriptionVideo = Subscription(1 << iota)
	SubscriptionAudio
	SubscriptionSubtitle
	SubscriptionMetadata
	SubscriptionAll = Subscription(0)
)

// Possible packet types
const (
	PacketVideo = iota + 1
	PacketAudio
	PacketSubtitle
	PacketMetadata
	PacketPause
	PacketStop
)

const (
	StateStopped = iota + 1
	StatePaused
	StatePlaying
	StateTransitioning
)

var AllStates = []int{StateStopped, StatePaused, StatePlaying, StateTransitioning}

type WebsocketControl struct {
	ID           string `json:"id"`
	Subscription uint32 `json:"subscription"`
}

// IsSubscribedTo returns whether or not the client subscription is subscribed
// to the given subscription.
func (s Subscription) IsSubscribedTo(sub Subscription) bool {
	return (s & sub) == sub
}

// Client is a websocket connected client.
type Client struct {
	mutex         *sync.Mutex
	id            string
	conn          *websocket.Conn
	subscriptions Subscription
}

// StreamManager represents a stream manager.
type StreamManager struct {
	clientsMutex *sync.Mutex
	clients      []*Client

	stateCond *sync.Cond
	state     State

	targetState uint32

	baseOptions juroku.EncoderOptions
}

type State struct {
	Title         string
	State         int
	Timestamp     time.Duration
	RelativeStart time.Time
	Duration      time.Duration
	Context       context.Context
	Cancel        func()
}

func (s *State) MarshalJSON() ([]byte, error) {
	type stateJSON struct {
		Title         string `json:"title"`
		State         int    `json:"state"`
		Timestamp     int    `json:"timestamp"`
		RelativeStart int64  `json:"relativeStart"`
		Duration      int    `json:"duration"`
	}

	timestamp := s.Timestamp.Seconds() - 1.5
	relativeStart := s.RelativeStart.Add(time.Duration(-1500 * time.Millisecond))

	return json.Marshal(stateJSON{
		Title:         s.Title,
		State:         s.State,
		Timestamp:     int(timestamp),
		RelativeStart: relativeStart.Unix(),
		Duration:      int(math.Round(s.Duration.Seconds())),
	})
}

func NewEmptyState() State {
	return State{
		Title:         "",
		State:         0,
		Timestamp:     -1,
		RelativeStart: time.Time{},
		Duration:      -1,
		Context:       nil,
		Cancel:        nil,
	}
}

func (s *State) UpdateWith(new State) {
	if new.Title != "" {
		s.Title = new.Title
	}
	if new.State != 0 {
		s.State = new.State
	}
	if new.Timestamp >= 0 {
		s.Timestamp = new.Timestamp
	}
	if !new.RelativeStart.IsZero() {
		s.RelativeStart = new.RelativeStart
	}
	if new.Duration >= 0 {
		s.Duration = new.Duration
	}
	if new.Context != nil {
		s.Context = new.Context
	}
	if new.Cancel != nil {
		s.Cancel = new.Cancel
	}
}

type Metadata struct {
	Title    string
	Duration time.Duration
}

func (s *StreamManager) Broadcast(sub Subscription, data ...[]byte) {
	s.clientsMutex.Lock()
	clientCopy := make([]*Client, len(s.clients))
	copy(clientCopy, s.clients)
	s.clientsMutex.Unlock()

	for _, client := range clientCopy {
		client.mutex.Lock()
		if client.subscriptions.IsSubscribedTo(sub) {
			for _, d := range data {
				client.conn.WriteMessage(websocket.BinaryMessage, d)
			}
		}
		client.mutex.Unlock()
	}
}

func (s *StreamManager) HandleConn(conn *websocket.Conn) {
	s.clientsMutex.Lock()
	client := &Client{
		mutex:         new(sync.Mutex),
		conn:          conn,
		subscriptions: 0,
	}
	s.clients = append(s.clients, client)
	s.clientsMutex.Unlock()

	defer func() {
		s.clientsMutex.Lock()
		defer s.clientsMutex.Unlock()

		for i, c := range s.clients {
			if c == client {
				s.clients = append(s.clients[:i], s.clients[i+1:]...)
				return
			}
		}
	}()

	for {
		msgType, data, err := client.conn.ReadMessage()
		if err != nil {
			log.Println("client disconnected:", err)
			return
		}

		if msgType != websocket.BinaryMessage && msgType != websocket.TextMessage {
			continue
		}

		log.Println("received message:", string(data))

		var controlMsg WebsocketControl
		err = json.Unmarshal(data, &controlMsg)
		if err != nil {
			log.Println("failed to unmarshal control message:", err)
			continue
		}

		if Subscription(controlMsg.Subscription).IsSubscribedTo(SubscriptionMetadata) {
			state := s.State()
			d, err := state.MarshalJSON()
			if err == nil {
				client.mutex.Lock()
				client.conn.WriteMessage(websocket.BinaryMessage, append([]byte{PacketMetadata}, d...))
				client.mutex.Unlock()
			} else {
				log.Println("juroku video: HandleConn: error encoding state JSON:", err)
			}
		}

		client.mutex.Lock()
		client.id = controlMsg.ID
		client.subscriptions = Subscription(controlMsg.Subscription)
		client.mutex.Unlock()
	}
}

func NewStreamManager(baseOptions juroku.EncoderOptions) *StreamManager {
	return &StreamManager{
		clientsMutex: new(sync.Mutex),
		stateCond:    sync.NewCond(new(sync.Mutex)),
		targetState:  StateStopped,
		state: State{
			State: StateStopped,
		},
		baseOptions: baseOptions,
	}
}

func (s *StreamManager) WaitForState(state int, ctx context.Context) (State, bool) {
	wrappedCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	done := false

	go func() {
		<-wrappedCtx.Done()
		s.stateCond.L.Lock()
		defer s.stateCond.L.Unlock()

		if !done {
			s.stateCond.Broadcast()
		}
	}()

	var matchedState State

	s.stateCond.L.Lock()
	defer s.stateCond.L.Unlock()

	for {
		s.stateCond.Wait()
		if s.state.State == state {
			matchedState = s.state
			break
		}
		if wrappedCtx.Err() != nil {
			return State{}, false
		}
	}

	done = true

	return matchedState, true
}

func (s *StreamManager) UpdateState(state State, requiredStates []int) bool {
	s.stateCond.L.Lock()

	matched := false
	for _, required := range requiredStates {
		if s.state.State == required {
			matched = true
		}
	}

	if !matched {
		s.stateCond.L.Unlock()
		return false
	}

	prevState := s.state
	s.state.UpdateWith(state)
	req, _ := state.MarshalJSON()
	now, _ := s.state.MarshalJSON()
	log.Println("state updated to:", string(now), "using", string(req))
	s.stateCond.Broadcast()
	newState := s.state
	s.stateCond.L.Unlock()

	d, err := newState.MarshalJSON()
	if err == nil {
		s.Broadcast(SubscriptionAll, append([]byte{PacketMetadata}, d...))
	} else {
		log.Println("juroku video: error encoding state JSON:", err)
	}

	if newState.State == StatePaused && prevState.State == StatePlaying {
		s.Broadcast(SubscriptionAll, []byte{PacketPause})
	} else if newState.State == StateStopped && (prevState.State == StatePlaying || prevState.State == StateStopped) {
		s.Broadcast(SubscriptionAll, []byte{PacketStop})
	}

	return true
}
