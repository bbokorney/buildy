package main

import "log"

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
}

func (b *Builder) run() {
	log.Println("Builder started")
	for {
		select {
		case req := <-b.buildReqChan:
			log.Printf("Build request received: %v\n", req)
			req.respChan <- BuildResult{true, "somehash"}
		}
	}
}
