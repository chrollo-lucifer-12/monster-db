package server

import "time"

type TimeProc func(loop *EventLoop, id int64, clientData interface{}) int

type TimeEvent struct {
	ID         int64
	When       time.Time
	Proc       TimeProc
	ClientData interface{}
}

func (loop *EventLoop) processTimeEvents() {
	now := time.Now()
	var validTimeEvents []*TimeEvent

	for _, te := range loop.TimeEvents {
		if now.After(te.When) {
			retval := te.Proc(loop, te.ID, te.ClientData)
			if retval != -1 {
				te.When = time.Now().Add(time.Duration(retval) * time.Millisecond)
				validTimeEvents = append(validTimeEvents, te)
			}
		} else {
			validTimeEvents = append(validTimeEvents, te)
		}
	}
	loop.TimeEvents = validTimeEvents
}

func (el *EventLoop) AddTimeEvent(ms int64, proc TimeProc, clientData interface{}) int64 {
	id := el.NextTimeEventID
	el.NextTimeEventID++

	te := &TimeEvent{
		ID:         id,
		When:       time.Now().Add(time.Duration(ms) * time.Millisecond),
		Proc:       proc,
		ClientData: clientData,
	}

	el.TimeEvents = append(el.TimeEvents, te)
	return id
}
