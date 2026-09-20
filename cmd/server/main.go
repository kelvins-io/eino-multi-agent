package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/kelvins-io/eino-multi-agent/internal/agent"
	"github.com/kelvins-io/eino-multi-agent/internal/api"
	"github.com/kelvins-io/eino-multi-agent/internal/config"
	"github.com/kelvins-io/eino-multi-agent/internal/connector"
	"github.com/kelvins-io/eino-multi-agent/internal/harness"
	"github.com/kelvins-io/eino-multi-agent/internal/logx"
	"github.com/kelvins-io/eino-multi-agent/internal/scheduler"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "", "path to config.yaml")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		// logger not ready yet
		_, _ = os.Stderr.WriteString("load config: " + err.Error() + "\n")
		os.Exit(1)
	}
	if _, err := logx.Init(cfg.Server.LogLevel, cfg.Server.LogFormat, cfg.Server.LogFile, cfg.Server.LogKeepDays); err != nil {
		_, _ = os.Stderr.WriteString("init logger: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer logx.Sync()

	log := logx.Named("server")
	st, err := store.Open(cfg.Database)
	if err != nil {
		log.Fatal("open database", zap.Error(err))
	}
	skills, err := agent.LoadSkills(cfg.Skills.Dir)
	if err != nil {
		log.Fatal("load skills", zap.Error(err))
	}

	var factory *agent.Factory
	if err := agent.Ready(cfg.LLM); err == nil {
		factory, err = agent.NewFactory(context.Background(), cfg)
		if err != nil {
			log.Warn("llm factory not ready", zap.Error(err))
		} else {
			log.Info("llm ready",
				zap.String("provider", cfg.LLM.Provider),
				zap.String("model", cfg.LLM.Model),
			)
		}
	} else {
		log.Warn("llm not configured yet", zap.Error(err))
	}

	rt := harness.NewRuntime(cfg, st, factory, skills)
	reg := connector.New(st, cfg.Workspace.Root)
	rt.SetOnSuccess(func(ctx context.Context, task *store.Task, arts []store.Artifact) {
		reg.Notify(ctx, connector.Event{Task: task, Artifacts: arts})
	})
	sched := scheduler.New(st, rt, 20*time.Second)
	sched.Start(context.Background())
	rt.Recover(context.Background())
	srv := api.New(cfg, st, rt, sched, reg)
	log.Info("work harness listening",
		zap.String("addr", cfg.Server.Addr),
		zap.String("mode", cfg.Server.Mode),
		zap.String("log_level", cfg.Server.LogLevel),
		zap.String("log_file", cfg.Server.LogFile),
		zap.Int("log_keep_days", cfg.Server.LogKeepDays),
		zap.Int("skills", len(skills)),
	)
	if err := srv.Engine().Run(cfg.Server.Addr); err != nil {
		log.Fatal("http server stopped", zap.Error(err))
	}
}
