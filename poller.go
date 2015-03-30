package main

import (
	"encoding/json"
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

func (bp *BranchPoller) poll() (modified bool) {
	urlStr := fmt.Sprintf("https://api.github.com/%v/%v/DBI/commits?per_page=1&sha=%v",
		bp.branchInfo.user, bp.branchInfo.repo, bp.branchInfo.branchName)
	u, err := url.Parse()
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
	client := http.Client{}
	resp, err := client.Do(&req)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(resp)
	// data, _ := ioutil.ReadAll(resp.Body)
	// fmt.Println(string(data))
	var data []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
	// read the Last-Modified header value
	fmt.Println(resp.Header["Last-Modified"])
	fmt.Println(resp.Header["Last-Modified"][0])
	req.Header["If-Modified-Since"] = []string{resp.Header["Last-Modified"][0]}
	client = http.Client{}
	resp, err = client.Do(&req)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(resp)

	return true
}

func (bp *BranchPoller) run() {
	log.Printf("Starting polling on branch %v\n", bp.branchInfo.branchName)
	bp.lastModified = ""
	// poll on the branch forerver
	for {
		if bp.poll() {
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
