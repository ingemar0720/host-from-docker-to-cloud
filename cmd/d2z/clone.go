package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
)

func runClone(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("clone", flag.ContinueOnError)
	repo := fs.String("repo", "", "git repository URL (or set REPO_URL)")
	dir := fs.String("dir", "", "clone target directory (or set WORK_DIR)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	u := *repo
	if u == "" {
		u = os.Getenv("REPO_URL")
	}
	if u == "" {
		return fmt.Errorf("clone requires -repo or REPO_URL")
	}
	target := *dir
	if target == "" {
		target = os.Getenv("WORK_DIR")
	}
	if target == "" {
		return fmt.Errorf("clone requires -dir or WORK_DIR")
	}
	if _, err := os.Stat(target); err == nil {
		cmd := exec.CommandContext(ctx, "git", "-C", target, "pull", "--ff-only")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	cmd := exec.CommandContext(ctx, "git", "clone", u, target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
