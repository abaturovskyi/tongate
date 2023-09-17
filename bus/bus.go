package bus

import (
	"context"
	"fmt"
	"time"

	"github.com/croutondefi/bus/v3"
	"github.com/mustafaturan/monoton/v3"
	"github.com/mustafaturan/monoton/v3/sequencer"
)

// Bus is a ref to bus.Bus
var Bus *bus.Bus

// Monoton is an instance of monoton.Monoton
var Monoton monoton.Monoton

// Init inits the bus config
func InitBus() {
	// configure id generator (it doesn't have to be monoton)
	node := uint64(1)
	initialTime := uint64(1577865600000)
	m, err := monoton.New(sequencer.NewMillisecond(), node, initialTime)
	if err != nil {
		panic(err)
	}

	// init an id generator
	var idGenerator bus.Next = m.Next

	// create a new bus instance
	b, err := bus.NewBus(idGenerator, 5)
	if err != nil {
		panic(err)
	}

	Bus = b
	Monoton = m
}

func RegisterHandler(key string, h bus.Handler) {
	Bus.RegisterHandler(key, h)

	fmt.Printf("Registered %s handler.\n", key)
}

func EmitEvent(topic string, q any) error {
	return emitEvent(topic, q)
}

func emitEvent(topic string, q any) error {
	ctx := context.WithValue(context.Background(), bus.CtxKeyTxID, Monoton.Next())

	go func() {
		time.Sleep(2 * time.Second)
		Bus.Emit(ctx, topic, q)
	}()
	return nil
}
