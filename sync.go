package gomatrix

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"
)

// Syncer represents an interface that must be satisfied in order to do /sync requests on a client.
type Syncer interface {
	// ProcessResponse process the /sync response. The since parameter is the since= value that was used to produce the response.
	// This is useful for detecting the very first sync (since=""). If an error is return, Syncing will be stopped
	// permanently.
	ProcessResponse(resp *RespSync, since string) error
	// OnFailedSync returns either the time to wait before retrying or an error to stop syncing permanently.
	OnFailedSync(res *RespSync, err error) (time.Duration, error)
	// GetFilterJSON for the given user ID. NOT the filter ID.
	GetFilterJSON(userID string) json.RawMessage
}

// DefaultSyncer is the default syncing implementation. You can either write your own syncer, or selectively
// replace parts of this default syncer (e.g. the ProcessResponse method). The default syncer uses the observer
// pattern to notify callers about incoming events. See DefaultSyncer.OnEventType for more information.
type DefaultSyncer struct {
	UserID     string
	eventsChan chan<- *Event
}

// NewDefaultSyncer returns an instantiated DefaultSyncer
func NewDefaultSyncer(userID string, eventsChan chan<- *Event) *DefaultSyncer {
	return &DefaultSyncer{
		UserID:     userID,
		eventsChan: eventsChan,
	}
}

// ProcessResponse processes the /sync response in a way suitable for bots. "Suitable for bots" means a stream of
// unrepeating events. Returns a fatal error if a listener panics.
func (s *DefaultSyncer) ProcessResponse(res *RespSync, since string) (err error) {
	if !s.shouldProcessResponse(res, since) {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("ProcessResponse panicked! userID=%s since=%s panic=%s\n%s", s.UserID, since, r, debug.Stack())
		}
	}()

	for _, e := range res.AccountData.Events {
		s.eventsChan <- &e
	}
	for _, e := range res.Presence.Events {
		s.eventsChan <- &e
	}

	for roomID, roomData := range res.Rooms.Join {
		for _, event := range roomData.State.Events {
			event.RoomID = roomID
			s.eventsChan <- &event
		}
		for _, event := range roomData.Timeline.Events {
			event.RoomID = roomID
			s.eventsChan <- &event
		}
		for _, event := range roomData.Ephemeral.Events {
			event.RoomID = roomID
			s.eventsChan <- &event
		}
	}
	for roomID, roomData := range res.Rooms.Invite {
		for _, event := range roomData.State.Events {
			event.RoomID = roomID
			s.eventsChan <- &event
		}
	}
	for roomID, roomData := range res.Rooms.Leave {
		for _, event := range roomData.Timeline.Events {
			event.RoomID = roomID
			s.eventsChan <- &event
		}
	}
	return
}

// shouldProcessResponse returns true if the response should be processed. May modify the response to remove
// stuff that shouldn't be processed.
func (s *DefaultSyncer) shouldProcessResponse(resp *RespSync, since string) bool {
	if since == "" {
		return false
	}
	// This is a horrible hack because /sync will return the most recent messages for a room
	// as soon as you /join it. We do NOT want to process those events in that particular room
	// because they may have already been processed (if you toggle the bot in/out of the room).
	//
	// Work around this by inspecting each room's timeline and seeing if an m.room.member event for us
	// exists and is "join" and then discard processing that room entirely if so.
	// TODO: We probably want to process messages from after the last join event in the timeline.
	for roomID, roomData := range resp.Rooms.Join {
		for i := len(roomData.Timeline.Events) - 1; i >= 0; i-- {
			e := roomData.Timeline.Events[i]
			if e.Type == MemberEventType && e.StateKey != nil && *e.StateKey == s.UserID {
				m := e.Content["membership"]
				mship, ok := m.(string)
				if !ok {
					continue
				}
				if mship == "join" {
					_, ok = resp.Rooms.Join[roomID]
					if !ok {
						continue
					}
					delete(resp.Rooms.Join, roomID)   // don't re-process messages
					delete(resp.Rooms.Invite, roomID) // don't re-process invites
					break
				}
			}
		}
	}
	return true
}

// OnFailedSync always returns a 10 second wait period between failed /syncs, never a fatal error.
func (s *DefaultSyncer) OnFailedSync(res *RespSync, err error) (time.Duration, error) {
	return 10 * time.Second, nil
}

// GetFilterJSON returns a filter with a timeline limit of 50.
func (s *DefaultSyncer) GetFilterJSON(userID string) json.RawMessage {
	return json.RawMessage(`{"room":{"timeline":{"limit":50}}}`)
}
