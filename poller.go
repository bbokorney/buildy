package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
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

func (bp *BranchPoller) poll(req *http.Request) (modified bool, err error) {
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == http.StatusNotModified {
		return false, nil
	}
	if resp.StatusCode == http.StatusOK {
		val, ok := resp.Header["Last-Modified"]
		if ok {
			log.Printf("Setting header value to %v", val[0])
			req.Header["If-Modified-Since"] = []string{val[0]}
		}
		return true, nil
	} else {
		return false, fmt.Errorf("Unexpected status code: %v", resp.Status)
	}
}

func (bp *BranchPoller) run() {
	log.Printf("Starting polling on branch %v\n", bp.branchInfo.branchName)
	bp.lastModified = ""
	// poll on the branch forerver
	urlStr := fmt.Sprintf("https://api.github.com/repos/%v/%v/commits?per_page=1&sha=%v",
		bp.branchInfo.user, bp.branchInfo.repo, bp.branchInfo.branchName)
	u, err := url.Parse(urlStr)
	if err != nil {
		log.Printf("Error parsing url: %v", err)
		return
	}
	req := http.Request{
		Method: "GET",
		URL:    u,
		Header: map[string][]string{
			"Authorization": {fmt.Sprintf("token %v", bp.branchInfo.oauthtoken)},
		},
	}

	for {
		modified, err := bp.poll(&req)
		if err == nil {
			if modified {
				// the branch has changed, request a build of it
				log.Printf("Change detected in branch %v\n", bp.branchInfo.branchName)
				resultChan := make(chan BuildResult)
				bp.buildReqChan <- BuildRequest{bp.branchInfo.branchName, resultChan}
				result := <-resultChan
				passFail := "passed"
				if !result.pass {
					passFail = "failed"
				}
				log.Printf("Build for %v:%v %v!\n%v", bp.branchInfo.branchName,
					result.hash[:7], passFail, string(result.output))
				log.Printf("Post output: %v", string(result.postOutput))

			} else {
				log.Printf("No change detected in branch %v\n", bp.branchInfo.branchName)
			}
		} else {
			log.Printf("Error polling on branch %v: %v\n", bp.branchInfo.branchName, err)
			return
		}
		time.Sleep(bp.sleepDuration)
	}
}
