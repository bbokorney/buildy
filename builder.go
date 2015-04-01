package main

import (
    "fmt"
    "io/ioutil"
    "log"
    "os/exec"
    "strings"
)

type BuildRequest struct {
    branchName string
    respChan   chan BuildResult
}

type BuildResult struct {
    pass        bool
    hash        string
    output      []byte
    postSuccess bool
    postOutput  []byte
}

type Builder struct {
    buildReqChan chan BuildRequest
    path         string
    cmds         []Command
    postCmd      string
}

func (b *Builder) makeCmd(name string, args ...string) *exec.Cmd {
    cmd := exec.Command(name, args...)
    cmd.Dir = b.path
    return cmd
}

func (b *Builder) getCurrentCommitHash() (hash string) {
    cmd := b.makeCmd("git", "rev-parse", "HEAD")
    output, err := cmd.CombinedOutput()
    if err != nil {
        log.Printf("Error getting commit hash: %v", err)
        return "Invalid-hash"
    }
    return strings.TrimSpace(string(output))
}

func (b *Builder) executeCmds(cmds []*exec.Cmd) (output []byte, err error) {
    for _, c := range cmds {
        out, err := c.CombinedOutput()
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
            log.Printf("Build request received for branch %v\n", req.branchName)
            var cmds []*exec.Cmd = []*exec.Cmd{b.makeCmd("git", "checkout", req.branchName), b.makeCmd("git", "pull")}
            for _, c := range b.cmds {
                cmds = append(cmds, b.makeCmd(c.Name, c.Args...))
            }
            output, err := b.executeCmds(cmds)
            pass := true
            if err != nil {
                pass = false
            }

            hash := b.getCurrentCommitHash()
            // write output to file
            filename := fmt.Sprintf("%v-%v.txt", req.branchName, hash)
            err = ioutil.WriteFile(filename, output, 0644)
            postSuccess := true
            var postOutput []byte
            if err != nil {
                postSuccess = false
                log.Printf("Could not create build output file %v: %v", filename, err)
            } else {
                // execute the post processing program
                cmd := exec.Command(b.postCmd, filename)
                postOutput, err = cmd.CombinedOutput()
                if err != nil {
                    postSuccess = false
                }
            }
            req.respChan <- BuildResult{pass, hash, output, postSuccess, postOutput}
        }
    }
}
