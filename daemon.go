package errcat

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	errcatapi "github.com/agschwender/errcat-go/api"
)

const bufferSize = 100
const tickerDuration = time.Duration(15) * time.Second

// Daemon is the background processor that will collect all calls and
// send them to the errcat server.
type Daemon struct {
	addr    url.URL
	env     string
	service string

	callCh   chan errcatapi.Call
	client   errcatapi.Client
	ctx      context.Context
	cancelFn context.CancelFunc
	registry map[string]Caller
}

type optionD func(d *Daemon)

func NewD(opts ...optionD) *Daemon {
	d := &Daemon{
		callCh:   make(chan errcatapi.Call),
		registry: make(map[string]Caller),
	}
	for _, opt := range opts {
		opt(d)
	}

	return d
}

// WithClient defines the client that should be used for sending metrics
// to the errcat server. This allows finer control over the client than
// WithServerAddr.
func WithClient(client errcatapi.Client) optionD {
	return func(d *Daemon) {
		d.client = client
	}
}

// WithEnvironment defines the environment the daemon should indicate
// the calls are being made in.
func WithEnvironment(env string) optionD {
	return func(d *Daemon) {
		d.env = env
	}
}

// WithServerAddr will create a client for communicating to the errcat
// server using the supplied server address.
func WithServerAddr(addr url.URL) optionD {
	return func(d *Daemon) {
		d.addr = addr
	}
}

// WithService defines the service the daemon should indicate the
// calls are initiated from.
func WithService(service string) optionD {
	return func(d *Daemon) {
		d.service = service
	}
}

// RegisterCaller attaches a caller to the daemon so that it does not
// need to be re-instantiated.
func (d *Daemon) RegisterCaller(c Caller) (string, error) {
	if d == nil {
		return "", nil
	}

	if _, ok := d.registry[c.key]; ok {
		return c.key, fmt.Errorf(
			"caller has already been registered with the dependency and name of %q and %q",
			c.dependency,
			c.name,
		)
	}
	d.registry[c.key] = c
	return c.key, nil
}

// Call executes the supplied function using the caller looked up with
// the key.
func (d *Daemon) Call(key string, cb CallFn) (err error) {
	if d == nil {
		return cb()
	}

	caller := d.registry[key]

	call := errcatapi.Call{
		Dependency: caller.dependency,
		Name:       caller.name,
		StartedAt:  time.Now(),
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
		call.Error = err
		call.Duration = time.Now().Sub(call.StartedAt)
		if d.enabled() {
			d.callCh <- call
		}
	}()

	err = caller.Call(cb)
	return
}

func (d *Daemon) Start() {
	if d == nil {
		return
	}

	d.ctx, d.cancelFn = context.WithCancel(context.Background())
	go d.consumeCalls()
}

func (d *Daemon) Stop() {
	if d == nil {
		return
	}
	d.cancelFn()
}

func (d *Daemon) consumeCalls() {
	log.Printf("in consumeCalls")

	if !d.enabled() {
		log.Printf("errcat disabled")
		return
	}

	log.Printf("errcat enabled")

	calls := make([]errcatapi.Call, 0, bufferSize)

	ticker := time.NewTicker(tickerDuration)
	defer ticker.Stop()

	for {
		select {
		case call := <-d.callCh:
			log.Printf("received call")
			calls = append(calls, call)
			if len(calls) == bufferSize {
				log.Printf("flushing")
				d.send(calls)
				calls = calls[:0]
			}
		case <-ticker.C:
			d.send(calls)
			calls = calls[:0]
		case <-d.ctx.Done():
			d.send(calls)
			return
		}
	}
}

func (d *Daemon) enabled() bool {
	return d.client != nil || d.addr.Host != ""
}

func (d *Daemon) safeClient() errcatapi.Client {
	if d.client != nil {
		return d.client
	}

	d.client, _ = errcatapi.NewClient(d.addr)
	return d.client
}

func (d *Daemon) send(calls []errcatapi.Call) {
	if len(calls) == 0 {
		return
	}

	client := d.safeClient()
	if client == nil {
		// TODO(agschwender): retry buffer? logger error
		return
	}

	log.Printf("sending %d calls", len(calls))

	// TODO(agschwender): what to do with errors, same as above.
	err := client.RecordCalls(context.Background(), errcatapi.RecordCallsRequest{
		Environment: d.env,
		Service:     d.service,
		Calls:       calls,
	})
	if err != nil {
		log.Printf("record call failed: %v", err)
	}
}
