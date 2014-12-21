package main

import (
	"fmt"
	"strings"
)

type Command struct {
	Path		string
	Name		string
	Parameters	[]string
}

type CommandSet struct {
	Commands	[]Command
}

func (c *Command) String() string {
	return fmt.Sprintf("%s %s", c.Name, strings.Join(c.Parameters, " "))
}
