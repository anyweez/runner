package main

/**
 * This package pulls tasks from a Beanstalk queue and runs them on the local
 * system.
 */

import (
	"encoding/json"
	"io/ioutil"
	gproto "github.com/golang/protobuf/proto"
	"flag"
	"fmt"
	beanstalk "github.com/kr/beanstalk"
	"os"
	"os/exec"
	"os/signal"
	cr "proto"
	"log"
	"time"
)

var LIMIT = flag.Int("limit", 2, "The maximum number of jobs to run in parallel.")
var COMMAND_FILE = flag.String("commands", "data/commands.json", "A file describing the list of commands.")
var TARGET_QUEUE = flag.String("queue", "default", "The name of the queue that should be pulled from.")
var SOURCE = flag.String("source", "127.0.0.1:11300", "The network address of the beanstalkd instance.")

// Keep track of the number of jobs that are currently running.
var activeJobs = 0

/**
 * This function reads in a list of valid commands from an external
 * JSON file.
 */
func readCmds(filename string) map[string]Command {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(fmt.Sprintf("Command file '%s' does not exist.",  filename))
	}

	cs := CommandSet{}
	json.Unmarshal(buf, &cs)

	commands := make(map[string]Command)

	for _, command := range cs.Commands {
		commands[command.Name] = command
	}

	return commands
}

/**
 * Once the job has been validated, launch is kicked off on a separate
 * goroutine to launch the command as a separate process.
 */
func launch(c Command, finished chan bool) {	
	cmd := exec.Command(c.Path, c.Parameters...)
	err := cmd.Run()

	if err != nil {
		log.Println(fmt.Sprintf("Warning [%s]: %s", c.Name, err.Error()))
	}

	activeJobs -= 1

	// Job is finished so open up the slot again.
	finished <- true
//	conn.Delete(id)
}

/**
 * Check to make sure the requested command is registered in the command file.
 */
func loadCommand(c Command, commands map[string]Command) (Command, bool) {
	command, exists := commands[c.Name]

	if exists {
		c.Path = command.Path
	}

	return c, exists
}

/**
 * Transforms the wire format version of the command request into an internal
 * data structure.
 */
func transformRequest(request cr.CommandRequest) Command {
	c := Command{}

	c.Name = *request.Name
	c.Parameters = request.Params

	return c
}

/**
 * Listen for a SIGINT signal. If we detect one, stop accepting new
 * messages from the queue but finish launching anything that's already
 * been pulled off.
 */
func signals(running *bool, sig chan os.Signal) {
	for {
		// Check to see if a sigint has been received. If so, stop
		// accepting new jobs.
		select {
			case <- sig:
				*running = false
				log.Println("Received interrupt signal; finishing polling session.")
				return
			default:
				break
		}
	}
}

/**
 * Read in a list of eligible commands and execute each one in parallel (up to 
 * a limit of LIMIT).
 */
func main() {
	flag.Parse()

	// Stop accepting new commands and shutdown when SIGINT is received.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	running := true
	// Monitor the state of this channel in a separate goroutine.
	go signals(&running, sig)
	
	commands := readCmds(*COMMAND_FILE)
	available := make(chan bool, *LIMIT)

	for i := 0; i < *LIMIT; i++ {
		available <- true
	}

	// Continue pulling commands from the queue whenever something
	// is available.
	conn, err := beanstalk.Dial("tcp", *SOURCE)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println("Connected to beanstalkd at " + *SOURCE)
	
	// Set the connection's tube to make sure we're pulling the desired
	// jobs.
	conn.Tube = beanstalk.Tube{
		Conn: conn,
		Name: *TARGET_QUEUE,
	}
	
	conn.TubeSet = *beanstalk.NewTubeSet(conn, *TARGET_QUEUE)
	
	for {
		// If we're received a sigint, we should end the process gracefully.
		if !running {
			log.Println("Shutting down runner...")
			return
		}
		
		request := cr.CommandRequest{}
		id, body, err := conn.Reserve(2 * time.Second)
		
		// If we didn't get an actual command, continue polling.
		if err != nil {
			continue
		}
		
		gproto.Unmarshal(body, &request)
		command := transformRequest(request)

		cmd, valid := loadCommand(command, commands)
		if valid {
			// Wait for an open job slot to be available.
			<-available
			log.Println(fmt.Sprintf("Executing requested command '%s'", cmd.String()))
			go launch(cmd, available)
			
			conn.Delete(id)
			activeJobs += 1
		} else {
			log.Println(fmt.Sprintf("Invalid command requested: '%s'", cmd.String()))
		}
	}
}
