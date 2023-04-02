package main

import (
	"context"
	"errors"
	"log"
	"net/url"
	"time"

	errcatapi "github.com/agschwender/errcat-go/api"
)

func main() {
	errcatAddr, err := url.Parse("localhost:8000")
	if err != nil {
		log.Fatalf("bad url: %v", err)
	}

	log.Printf("Connecting to %s", errcatAddr.String())

	c, err := errcatapi.NewClient(*errcatAddr)
	if err != nil {
		log.Fatalf("unable to connect to errcat server: %v", err)
	}
	defer c.Close()

	err = c.RecordCalls(context.Background(), errcatapi.RecordCallsRequest{
		Environment: "dev",
		Calls: []errcatapi.Call{
			{
				Dependency: "mysql",
				Duration:   time.Duration(125) * time.Millisecond,
				Error:      errors.New("oops"),
				Name:       "foo.getName",
				StartedAt:  time.Now().Add(-1 * time.Duration(125) * time.Millisecond),
			},
			{
				Dependency: "google",
				Duration:   time.Duration(375) * time.Millisecond,
				Error:      nil,
				Name:       "google.Search",
				StartedAt:  time.Now().Add(-1 * time.Duration(375) * time.Millisecond),
			},
		},
	})
	if err != nil {
		log.Printf("encountered error when recording calls: %v", err)
	}
}
