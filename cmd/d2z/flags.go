package main

import (
	"flag"
	"fmt"
	"strings"
)

type commonFlags struct {
	Workdir      string
	ComposeFiles []string
	StrategyFile string
}

func parseCommon(fs *flag.FlagSet, cf *commonFlags) {
	fs.StringVar(&cf.Workdir, "workdir", ".", "project directory (compose-relative paths resolve here)")
	fs.Func("f", "compose file path (repeatable); default: auto-detect in workdir", func(s string) error {
		cf.ComposeFiles = append(cf.ComposeFiles, s)
		return nil
	})
	fs.StringVar(&cf.StrategyFile, "strategy", "", "path to zeabur.strategy.yaml (optional)")
}

func resolveCompose(cf *commonFlags) []string {
	if len(cf.ComposeFiles) > 0 {
		return cf.ComposeFiles
	}
	return defaultComposeFiles(cf.Workdir)
}

func splitEnvList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseOptionalTools(s string) (zeabur, helm, bw bool, err error) {
	for _, p := range splitEnvList(s) {
		switch strings.ToLower(p) {
		case "zeabur":
			zeabur = true
		case "helm":
			helm = true
		case "bw", "bitwarden":
			bw = true
		default:
			return false, false, false, fmt.Errorf("unknown optional tool %q (use zeabur,helm,bw)", p)
		}
	}
	return zeabur, helm, bw, nil
}
