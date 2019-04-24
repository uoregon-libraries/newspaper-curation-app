package main

import (
	"strings"
)

type handler func(args []string)
type command struct {
	command string
	help    string
	handler handler
}
type menu struct {
	commands []*command
}

func (m *menu) add(cmd, help string, h handler) {
	m.commands = append(m.commands, &command{command: cmd, help: help, handler: h})
}

func (m *menu) dispatch(input string) bool {
	var fields = strings.Fields(input)
	if len(fields) < 1 {
		return false
	}
	var cmd, args = fields[0], fields[1:]

	for _, c := range m.commands {
		if strings.ToUpper(c.command) == strings.ToUpper(cmd) {
			c.handler(args)
			return true
		}
	}

	return false
}
