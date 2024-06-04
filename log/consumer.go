package log

import (
	"fmt"
	"time"
)

const StoppedForOutOfSyncMessage = "Stopping log consumer: Headers out of sync"

// logConsumerInterface {

// Consumer represents any object that can
// handle a Log, it is up to the Consumer instance
// what to do with the log
type Consumer interface {
	Accept(Log)
}

// }

// multiLogConsumer {
// MultiConsumer is a Consumer that can accept multiple Consumers
type MultiConsumer struct {
	Consumers []Consumer
}

// Accept sends the log to all the consumers
func (mc *MultiConsumer) Accept(l Log) {
	for _, c := range mc.Consumers {
		c.Accept(l)
	}
}

// }

// ConsumerConfig is a configuration object for the producer/consumer pattern
type ConsumerConfig struct {
	Opts     []ProductionOption // options for the production of logs
	Consumer Consumer           // consumer for the logs
}

type OptionsContainer interface {
	WithLogProductionTimeout(timeout time.Duration)
}

// ProductionOption is a functional option that can be used to configure the log production
type ProductionOption func(OptionsContainer)

// WithProductionTimeout is a functional option that sets the timeout for the log production.
// If the timeout is lower than 5s or greater than 60s it will be set to 5s or 60s respectively.
func WithProductionTimeout(timeout time.Duration) ProductionOption {
	return func(c OptionsContainer) {
		c.WithLogProductionTimeout(timeout)
	}
}

// exampleLogConsumer {

// StdoutConsumer is a LogConsumer that prints the log to stdout
type StdoutConsumer struct{}

// Accept prints the log to stdout
func (lc *StdoutConsumer) Accept(l Log) {
	fmt.Print(string(l.Content))
}

// }
