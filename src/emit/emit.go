package main

import (
	beanstalk "github.com/kr/beanstalk"
	"flag"
	"log"
	gproto "github.com/golang/protobuf/proto"
	"os"	
	"time"
	cr "proto"
)

var TARGET_QUEUE = flag.String("queue", "default", "The name of the queue that should be pulled from.")
var SOURCE = flag.String("source", "127.0.0.1:11300", "The network address of the beanstalkd instance.")

func main() {
	fullArgs := os.Args[1:]
	
	// Find the divider between emit params and params to pass through.
	fwdArgs := make([]string, 0)
	fwdCmd := ""
	for i, cmd := range fullArgs {
		if cmd == "--" {
			fwdCmd = fullArgs[i+1]
			fwdArgs = fullArgs[i+2:]
		}
	}
	
	if fwdCmd == "" {
		log.Fatal("No command to forward found.")
	}
	
	// Form a CommandRequest and send it off!
	cr := cr.CommandRequest{
		Name: gproto.String(fwdCmd),
		Params: fwdArgs,
	}
	
	// Connect to beanstalk and send the request along.
	data, _ := gproto.Marshal(&cr)
	conn, err := beanstalk.Dial("tcp", *SOURCE)
	
	if err != nil {
		log.Fatal("Couldn't connect to source queue: " + err.Error())
	}
	
	// Set the connection's tube to make sure we're pushing to a queue
	// where consumers will know how to handle it.
	tube := beanstalk.Tube{
		Conn: conn,
		Name: *TARGET_QUEUE,
	}

	if err != nil {
		log.Fatal("Couldn't establish connection to beanstalk instance: " + err.Error())
	}
	
	_, err = tube.Put(data, 1, 0, 2 * time.Hour)
	
	if err != nil {
		log.Fatal(err)
	}
}
