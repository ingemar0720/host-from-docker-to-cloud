package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/ingemar0720/host-from-docker-to-cloud/internal/composeproj"
	"github.com/ingemar0720/host-from-docker-to-cloud/internal/precheck"
	"github.com/ingemar0720/host-from-docker-to-cloud/internal/sourcemode"
)

func runCheck(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	var cf commonFlags
	var optional string
	fs.StringVar(&optional, "optional", "", "comma-separated optional tools to verify: zeabur,helm,bw")
	parseCommon(fs, &cf)
	if err := fs.Parse(args); err != nil {
		return err
	}
	zb, helm, bw, err := parseOptionalTools(optional)
	if err != nil {
		return err
	}
	pre := precheck.Run(precheck.Options{
		CheckZeabur: zb, CheckHelm: helm, CheckBW: bw,
	})
	files := resolveCompose(&cf)
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			pre.Issues = append(pre.Issues, precheck.Issue{Name: "compose file", Detail: fmt.Sprintf("%s: %v", f, err)})
		}
	}
	proj, err := composeproj.Load(ctx, cf.Workdir, files)
	if err != nil {
		pre.Issues = append(pre.Issues, precheck.Issue{Name: "compose load", Detail: err.Error()})
	} else {
		for _, vErr := range sourcemode.Validate(proj) {
			pre.Issues = append(pre.Issues, precheck.Issue{Name: "source mode", Detail: vErr.Error()})
		}
	}
	if !pre.OK() {
		return fmt.Errorf("check failed:\n%s", pre.Error())
	}
	fmt.Println("d2z check: ok (tools + compose load + no depends_on cycles + explicit source mode)")
	return nil
}
