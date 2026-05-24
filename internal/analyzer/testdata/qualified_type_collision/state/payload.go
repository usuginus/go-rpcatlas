package state

import (
	"context"

	"example.com/app/sender"
)

type Payload interface {
	Token() string
	Advance(context.Context) Payload
}

type Events interface {
	Send(context.Context)
}

type Worker struct {
	events Events
}

func (w *Worker) Run(ctx context.Context) {
	w.events.Send(ctx)
}

var _ Events = (*sender.Sender)(nil)
