package main

import (
	"context"
	"flag"
	"os"

	"github.com/ingemar0720/host-from-docker-to-cloud/internal/composeproj"
	"github.com/ingemar0720/host-from-docker-to-cloud/internal/strategy"
	"github.com/ingemar0720/host-from-docker-to-cloud/internal/zeabur"
)

func runRender(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	var cf commonFlags
	outPath := fs.String("out", "", "write YAML to file (default: stdout)")
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
	classes := map[string]strategy.Result{}
	for _, name := range proj.ServiceNames() {
		classes[name] = strategy.Classify(sf, name, proj.Services[name])
	}
	out := os.Stdout
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}
	return zeabur.Render(ctx, out, proj, classes)
}
