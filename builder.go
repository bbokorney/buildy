package main

import (
	"log"
	"os/exec"
)

type BuildRequest struct {
	branchName string
	respChan   chan BuildResult
}

type BuildResult struct {
	pass bool
	hash string
}

type Builder struct {
	buildReqChan chan BuildRequest
	path         string
	cmds         []Command
}

func (b *Builder) makeCmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Dir = b.path
	return cmd
}

func (b *Builder) getCurrentCommitHash() (hash string) {
	cmd := b.makeCmd("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	log.Println(string(output))
	if err != nil {
		log.Printf("Error getting commit hash: %v", err)
		return "Invalid-hash"
	}
	return string(output)
}

func (b *Builder) executeCmds(cmds []*exec.Cmd) error {
	for _, c := range cmds {
		err := c.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Builder) run() {
	log.Println("Builder started")

	var commands []*exec.Cmd
	commands = append(commands, b.makeCmd("git", "pull"))
	for _, c := range b.cmds {
		commands = append(commands, b.makeCmd(c.Name, c.Args...))
	}

	for {
		select {
		case req := <-b.buildReqChan:
			log.Printf("Build request received: %v\n", req)
			var cmds []*exec.Cmd
			cmds = append(commands, b.makeCmd("git", "checkout", req.branchName))
			err := b.executeCmds(cmds)
			hash := b.getCurrentCommitHash()
			if err == nil {
				req.respChan <- BuildResult{true, hash}
			} else {
				req.respChan <- BuildResult{false, hash}
			}
		}
	}
}
