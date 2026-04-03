package precheck

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Issue describes a failed prerequisite.
type Issue struct {
	Name   string
	Detail string
}

// Result aggregates prerequisite checks.
type Result struct {
	Issues []Issue
}

func (r *Result) OK() bool { return len(r.Issues) == 0 }

func (r *Result) Error() string {
	var b strings.Builder
	for _, i := range r.Issues {
		b.WriteString(fmt.Sprintf("- %s: %s\n", i.Name, i.Detail))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// Options toggles optional tools.
type Options struct {
	CheckAWS    bool
	CheckZeabur bool
	CheckHelm   bool
	CheckBW     bool
}

// Run checks required and optional CLI tools.
func Run(opt Options) Result {
	var r Result
	requireTool(&r, "git", "git --version")
	requireTool(&r, "docker", "docker version")
	requireDockerCompose(&r)
	if opt.CheckAWS {
		requireTool(&r, "aws", "aws --version")
	}
	if opt.CheckZeabur {
		requireTool(&r, "zeabur", "zeabur version")
	}
	if opt.CheckHelm {
		requireTool(&r, "helm", "helm version")
	}
	if opt.CheckBW {
		requireTool(&r, "bw", "bw --version")
	}
	return r
}

func requireTool(r *Result, name, checkCmd string) {
	if _, err := exec.LookPath(name); err != nil {
		r.Issues = append(r.Issues, Issue{Name: name, Detail: "not found in PATH"})
		return
	}
	args := strings.Fields(checkCmd)
	if len(args) < 1 {
		return
	}
	cmd := exec.Command(args[0], args[1:]...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		r.Issues = append(r.Issues, Issue{Name: name, Detail: fmt.Sprintf("%v (%s)", err, strings.TrimSpace(stderr.String()))})
	}
}

func requireDockerCompose(r *Result) {
	if _, err := exec.LookPath("docker"); err != nil {
		return // already reported
	}
	cmd := exec.Command("docker", "compose", "version")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		r.Issues = append(r.Issues, Issue{Name: "docker compose", Detail: fmt.Sprintf("%v (%s)", err, strings.TrimSpace(stderr.String()))})
	}
}
