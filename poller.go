package main

import (
	"log"
	"time"
)

type BranchInfo struct {
	branchName string
	user       string
	repo       string
	oauthtoken string
	emails     []string
}

type BranchPoller struct {
	branchInfo    BranchInfo
	lastModified  string
	buildReqChan  chan BuildRequest
	sleepDuration time.Duration
}

func poll() (modified bool) {
	return true
}

func (bp *BranchPoller) pollBranch() {
	log.Printf("Starting polling on branch %v\n", bp.branchInfo.branchName)
	bp.lastModified = ""
	// poll on the branch forerver
	for {
		if poll() {
			// the branch has changed, request a build of it
			log.Printf("Change detected in branch %v\n", bp.branchInfo.branchName)
			resultChan := make(chan BuildResult)
			bp.buildReqChan <- BuildRequest{bp.branchInfo.branchName, resultChan}
			result := <-resultChan
			log.Printf("Build result for branch %v: %v\n", bp.branchInfo.branchName, result)
		}
		time.Sleep(bp.sleepDuration)
	}
}
