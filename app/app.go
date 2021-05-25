package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	githubactions "github.com/sethvargo/go-githubactions"
	"github.com/vorteil/direktiv/pkg/model"
	yaml "gopkg.in/yaml.v2"
)

type args struct {
	name  string
	value string
}

const (
	serverIdx    = iota
	protocolIdx  = iota
	namespaceIdx = iota
	syncIdx      = iota
	forceIdx     = iota
	tokenIdx     = iota
)

func main() {

	in := []args{
		args{
			name: "server",
		},
		args{
			name: "protocol",
		},
		args{
			name: "namespace",
		},
		args{
			name: "sync",
		},
		args{
			name: "force",
		},
		args{
			name: "token",
		},
	}

	for i := range in {
		getValue(&in[i].value, in[i].name)
	}

	fmt.Printf("using server: %v\n", in[serverIdx].value)

	doSync(in)
}

func handleIndividual(in []args, path string) {

	githubactions.Infof("handling workflow %s\n", path)

	workflow, exists := getWorkflow(in, path, nil, true)
	workflow.Version = getRef()

	// update if necessary if it exists otherwise create it
	if exists {
		if hasChanges(path) {
			getWorkflow(in, path, workflow, false)
		}
	} else {
		githubactions.Infof("creating workflow %s in namespace %s\n", workflow.ID, in[namespaceIdx].value)
		getWorkflow(in, path, workflow, true)
	}

}

func doSync(in []args) {

	ref := getRef()
	githubactions.Infof("workgin ref %s\n", ref)

	path := in[syncIdx].value
	if strings.HasPrefix(path, "/") {
		path = fmt.Sprintf(".%s", path)
	}

	// check if file
	fi, err := os.Stat(path)
	if err != nil {
		githubactions.Fatalf("can not diff '%s': %v", path, err)
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():

		files, err := ioutil.ReadDir(path)
		if err != nil {
			githubactions.Fatalf("can not read directory '%s': %v", path, err)
		}

		for _, f := range files {
			p := fmt.Sprintf("%s/%s", path, f.Name())
			handleIndividual(in, p)
		}

	case mode.IsRegular():
		handleIndividual(in, path)
	}

}

func loadWorkflow(path string) *model.Workflow {

	wf, err := ioutil.ReadFile(path)
	if err != nil {
		githubactions.Fatalf("can not read workflow %s: %v", path, err)
	}

	// load workflow for id
	workflow := &model.Workflow{}
	err = workflow.Load(wf)
	if err != nil {
		githubactions.Fatalf("can not run load workflow %s: %v", path, err)
	}

	return workflow

}

func getWorkflow(in []args, path string, wfin *model.Workflow, create bool) (*model.Workflow, bool) {

	workflow := loadWorkflow(path)

	u := &url.URL{}
	u.Scheme = in[protocolIdx].value
	u.Host = in[serverIdx].value
	u.Path = fmt.Sprintf("/api/namespaces/%s/workflows/%s", in[namespaceIdx].value, workflow.ID)

	method := "GET"
	var data io.Reader
	if wfin != nil {

		wfb, err := yaml.Marshal(wfin)
		if err != nil {
			githubactions.Fatalf("can not marshal workflow: %v", err)
		}

		data = bytes.NewReader(wfb)
		if create {
			u.Path = fmt.Sprintf("/api/namespaces/%s/workflows", in[namespaceIdx].value)
			method = "POST"
		} else if wfin != nil {
			method = "PUT"
		}

	}

	githubactions.Infof("accessing workflow from %s\n", u.String())

	req, err := http.NewRequest(method, u.String(), data)
	if err != nil {
		githubactions.Fatalf("can not create request: %v", err)
	}

	req.Header.Set("Content-Type", "text/yaml")

	// set token if provided
	if len(in[tokenIdx].value) > 0 {
		githubactions.Infof("using token authentication\n")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", in[tokenIdx].value))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		githubactions.Fatalf("can not get workflow: %v", err)
	}

	githubactions.Infof("workflow request status code %d\n", resp.StatusCode)

	if resp.StatusCode == 200 {
		return workflow, true
	}

	return workflow, false

}

func getValue(val *string, key string) {
	*val = githubactions.GetInput(key)
}

func getRef() string {

	ref := runGit("rev-parse", "--short", "HEAD")
	if len(ref) == 0 {
		ref = os.Getenv("GITHUB_SHA")
	} else {
		ref = ref[0 : len(ref)-1]
	}

	return ref
}

func hasChanges(path string) bool {

	v := runGit("diff", "HEAD^", "HEAD", "--name-only", path)

	if len(v) > 0 {
		return true
	}

	return false

}

func runGit(cmds ...string) string {

	stderr := new(bytes.Buffer)
	stdout := new(bytes.Buffer)

	cmd := exec.Command("git", cmds...)
	cmd.Stderr = stderr
	cmd.Stdout = stdout

	err := cmd.Run()

	githubactions.Infof("git command: %v\nstdout: %v\nstderr: %v\n", cmd,
		strings.TrimSpace(string(stdout.Bytes())), strings.TrimSpace(string(stderr.Bytes())))

	if err != nil {
		githubactions.Fatalf("can not run git command %v: %v", cmd, err)
	}

	return string(stdout.Bytes())

}
