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
	pass   bool
	hash   string
	output []byte
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
	if err != nil {
		log.Printf("Error getting commit hash: %v", err)
		return "Invalid-hash"
	}
	return string(output)
}

func (b *Builder) executeCmds(cmds []*exec.Cmd) (output []byte, err error) {
	for _, c := range cmds {
		out, err := c.Output()
		output = append(output, out...)
		if err != nil {
			return output, err
		}
	}
	return output, nil
}

func (b *Builder) run() {
	log.Println("Builder started")

	for {
		select {
		case req := <-b.buildReqChan:
			log.Printf("Build request received: %v\n", req)
			var cmds []*exec.Cmd = []*exec.Cmd{b.makeCmd("git", "checkout", req.branchName), b.makeCmd("git", "pull")}
			for _, c := range b.cmds {
				cmds = append(cmds, b.makeCmd(c.Name, c.Args...))
			}
			output, err := b.executeCmds(cmds)
			hash := b.getCurrentCommitHash()
			if err == nil {
				req.respChan <- BuildResult{true, hash, output}
			} else {
				req.respChan <- BuildResult{false, hash, output}
			}
		}
	}
}
