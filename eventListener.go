package gomeassistant

import (
	"encoding/json"
	"time"

	"github.com/golang-module/carbon"
	"github.com/saml-dev/gome-assistant/internal"
	ws "github.com/saml-dev/gome-assistant/internal/websocket"
)

type EventListener struct {
	eventTypes   []string
	callback     EventListenerCallback
	betweenStart string
	betweenEnd   string
	throttle     time.Duration
	lastRan      carbon.Carbon

	exceptionDays   []time.Time
	exceptionRanges []timeRange
}

// TODO: add state object as second arg
type EventListenerCallback func(*Service, *State, EventData)

type EventData struct {
	Type         string
	RawEventJSON []byte
}

/* Methods */

func NewEventListener() eventListenerBuilder1 {
	return eventListenerBuilder1{EventListener{
		lastRan: carbon.Now().StartOfCentury(),
	}}
}

type eventListenerBuilder1 struct {
	eventListener EventListener
}

func (b eventListenerBuilder1) EventTypes(ets ...string) eventListenerBuilder2 {
	b.eventListener.eventTypes = ets
	return eventListenerBuilder2(b)
}

type eventListenerBuilder2 struct {
	eventListener EventListener
}

func (b eventListenerBuilder2) Call(callback EventListenerCallback) eventListenerBuilder3 {
	b.eventListener.callback = callback
	return eventListenerBuilder3(b)
}

type eventListenerBuilder3 struct {
	eventListener EventListener
}

func (b eventListenerBuilder3) OnlyBetween(start string, end string) eventListenerBuilder3 {
	b.eventListener.betweenStart = start
	b.eventListener.betweenEnd = end
	return b
}

func (b eventListenerBuilder3) OnlyAfter(start string) eventListenerBuilder3 {
	b.eventListener.betweenStart = start
	return b
}

func (b eventListenerBuilder3) OnlyBefore(end string) eventListenerBuilder3 {
	b.eventListener.betweenEnd = end
	return b
}

func (b eventListenerBuilder3) Throttle(s DurationString) eventListenerBuilder3 {
	d := internal.ParseDuration(string(s))
	b.eventListener.throttle = d
	return b
}

func (b eventListenerBuilder3) ExceptionDay(t time.Time) eventListenerBuilder3 {
	b.eventListener.exceptionDays = append(b.eventListener.exceptionDays, t)
	return b
}

func (b eventListenerBuilder3) ExceptionRange(start, end time.Time) eventListenerBuilder3 {
	b.eventListener.exceptionRanges = append(b.eventListener.exceptionRanges, timeRange{start, end})
	return b
}

func (b eventListenerBuilder3) Build() EventListener {
	return b.eventListener
}

type BaseEventMsg struct {
	Event struct {
		EventType string `json:"event_type"`
	} `json:"event"`
}

/* Functions */
func callEventListeners(app *App, msg ws.ChanMsg) {
	baseEventMsg := BaseEventMsg{}
	json.Unmarshal(msg.Raw, &baseEventMsg)
	listeners, ok := app.eventListeners[baseEventMsg.Event.EventType]
	if !ok {
		// no listeners registered for this event type
		return
	}

	for _, l := range listeners {
		// Check conditions
		if c := checkWithinTimeRange(l.betweenStart, l.betweenEnd); c.fail {
			continue
		}
		if c := checkThrottle(l.throttle, l.lastRan); c.fail {
			continue
		}
		if c := checkExceptionDays(l.exceptionDays); c.fail {
			continue
		}
		if c := checkExceptionRanges(l.exceptionRanges); c.fail {
			continue
		}

		eventData := EventData{
			Type:         baseEventMsg.Event.EventType,
			RawEventJSON: msg.Raw,
		}
		go l.callback(app.service, app.state, eventData)
		l.lastRan = carbon.Now()
	}
}
