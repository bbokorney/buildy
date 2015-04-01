package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Commit struct {
	Sha     string `json:"sha"`
	HtmlUrl string `json:"html_url"`
}

type BranchInfo struct {
	branchName string
	user       string
	repo       string
	oauthtoken string
}

type BranchPoller struct {
	branchInfo    BranchInfo
	email         EmailInfo
	lastModified  string
	buildReqChan  chan BuildRequest
	sleepDuration time.Duration
}

func (bp *BranchPoller) sendEmails(result BuildResult, commit *Commit) {
	passFail := "Success"
	passFailDescrip := "passed"
	if !result.pass {
		passFail = "Failure"
		passFailDescrip = "failed"
	}
	shortHash := result.hash[:7]
	shortName := fmt.Sprintf("%v:%v", bp.branchInfo.branchName, shortHash)
	subject := fmt.Sprintf("%v %v %v", bp.email.SubjectPrefix, passFail, shortName)
	text := fmt.Sprintf("Commit %v to branch %v has %v.\n", shortHash, bp.branchInfo.branchName, passFailDescrip)
	text += fmt.Sprintf("See the commit here:\n%v\n", commit.HtmlUrl)
	postOutput := string(result.postOutput)
	text += fmt.Sprintf("See the build output here:\n%v\n", postOutput)

	data := url.Values{"from": {bp.email.Sender}, "to": bp.email.Recipients, "subject": {subject}, "text": {text}}
	body := strings.NewReader(data.Encode())
	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.mailgun.net/v3/%v/messages", bp.email.MailGunDomain), body)
	if err != nil {
		log.Printf("Failed to build email request: %v\n", err)
		return
	}
	req.SetBasicAuth("api", bp.email.MailGunKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send email request: %v\n", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Email API returned error code %v", resp.StatusCode)
		strBody, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			log.Println(string(strBody))
		}
		return
	}
	log.Println("Emails successfully sent.")
}

func (bp *BranchPoller) getCommit(hash string) (commit *Commit, err error) {
	urlStr := fmt.Sprintf("https://api.github.com/repos/%v/%v/commits/%v",
		bp.branchInfo.user, bp.branchInfo.repo, hash)
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
	client := http.Client{}
	resp, err := client.Do(&req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusOK {
		var commit Commit
		err = json.NewDecoder(resp.Body).Decode(&commit)
		if err != nil {
			log.Printf("Could not decode commit: %v\n", err)
		} else {
			log.Printf("Commit: %v\n", commit)
		}
		return &commit, nil
	} else {
		return nil, fmt.Errorf("Unexpected status code getting commit: %v", resp.Status)
	}
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
		var commits []Commit
		err = json.NewDecoder(resp.Body).Decode(&commits)
		if err != nil {
			log.Printf("Could not decode commits: %v\n", err)
		} else {
			log.Printf("Commits: %v\n", commits)
		}
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
				commit, err := bp.getCommit(result.hash)
				if err != nil {
					commit = nil
					log.Printf("Error getting commit: %v\n", err)
				}
				log.Println("Sending emails")
				bp.sendEmails(result, commit)
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
