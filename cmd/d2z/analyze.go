package main

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/ingemar0720/host-from-docker-to-cloud/internal/composeproj"
	"github.com/ingemar0720/host-from-docker-to-cloud/internal/deployplan"
	"github.com/ingemar0720/host-from-docker-to-cloud/internal/strategy"
)

func runAnalyze(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	var cf commonFlags
	parseCommon(fs, &cf)
	if err := fs.Parse(args); err != nil {
		return err
	}
	sf, err := strategy.LoadStrategyFile(cf.StrategyFile)
	if err != nil {
		return err
	}
	if cf.StrategyFile == "" {
		def, err := strategy.LoadDefaultStrategyFile(cf.Workdir)
		if err != nil {
			return err
		}
		if len(def.Services) > 0 {
			sf = def
		}
	}
	proj, err := composeproj.Load(ctx, cf.Workdir, resolveCompose(&cf))
	if err != nil {
		return err
	}
	order, err := deployplan.Order(proj)
	if err != nil {
		return err
	}

	fmt.Printf("project: %q  workdir: %s\n\n", proj.Name, proj.WorkingDir)
	fmt.Println("deployment order:")
	for i, name := range order {
		fmt.Printf("  %d. %s\n", i+1, name)
	}
	fmt.Println()

	for _, name := range order {
		svc := proj.Services[name]
		r := strategy.Classify(sf, name, svc)
		fmt.Printf("service %q\n", name)
		fmt.Printf("  classification: %s (%s)\n", r.Kind, r.Reason)
		if svc.Image != "" {
			fmt.Printf("  image: %s\n", svc.Image)
		}
		if svc.Build != nil {
			fmt.Printf("  build: context=%s dockerfile=%s\n", svc.Build.Context, svc.Build.Dockerfile)
		}
		if len(svc.DependsOn) > 0 {
			fmt.Printf("  depends_on:\n")
			deps := make([]string, 0, len(svc.DependsOn))
			for dep := range svc.DependsOn {
				deps = append(deps, dep)
			}
			sort.Strings(deps)
			for _, dep := range deps {
				d := svc.DependsOn[dep]
				cond := strings.TrimSpace(d.Condition)
				if cond == "" {
					cond = types.ServiceConditionStarted
				}
				fmt.Printf("    - %s: condition=%q required=%v\n", dep, cond, d.Required)
			}
		}
		if svc.HealthCheck != nil && !svc.HealthCheck.Disable {
			hc := svc.HealthCheck
			fmt.Printf("  healthcheck: test=%v", hc.Test)
			if hc.Interval != nil {
				fmt.Printf(" interval=%s", hc.Interval.String())
			}
			if hc.Timeout != nil {
				fmt.Printf(" timeout=%s", hc.Timeout.String())
			}
			if hc.Retries != nil {
				fmt.Printf(" retries=%d", *hc.Retries)
			}
			fmt.Println()
		}
		fmt.Println()
	}
	return nil
}
