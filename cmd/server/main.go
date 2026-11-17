package main

import (
	"context"
	"flag"
	"net"
	"os"
	"strings"
	"time"

	"github.com/kelvins-io/eino-multi-agent/internal/a2a"
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
			if remotes, err := a2a.LoadRemoteAgents(context.Background(), cfg.A2A.Remotes); err != nil {
				log.Warn("a2a remotes not ready", zap.Error(err))
			} else if len(remotes) > 0 {
				factory.SetRemoteAgents(remotes)
			}
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

	ln, err := net.Listen("tcp", cfg.Server.Addr)
	if err != nil {
		log.Fatal("listen api", zap.String("addr", cfg.Server.Addr), zap.Error(err))
	}
	log.Info("work harness listening",
		zap.String("addr", cfg.Server.Addr),
		zap.String("mode", cfg.Server.Mode),
		zap.String("log_level", cfg.Server.LogLevel),
		zap.String("log_file", cfg.Server.LogFile),
		zap.Int("log_keep_days", cfg.Server.LogKeepDays),
		zap.Int("skills", len(skills)),
		zap.Bool("a2a", cfg.A2A.Enabled),
		zap.Bool("a2a_auto_register", cfg.A2A.AutoRegister),
	)

	// 主 API 监听成功后再拉起 A2A，并向注册中心登记
	if cfg.A2A.Enabled && cfg.A2A.AutoRegister {
		startAndRegisterA2A(cfg, rt, st, log)
	} else if cfg.A2A.Enabled {
		if strings.TrimSpace(cfg.A2A.AuthToken) == "" {
			log.Warn("a2a enabled without auth_token; inbound endpoints are open")
		}
		a2aSrv, err := a2a.NewServer(cfg, rt, st)
		if err != nil {
			log.Fatal("a2a server", zap.Error(err))
		}
		a2aSrv.Start(nil)
	}

	if err := srv.Engine().RunListener(ln); err != nil {
		log.Fatal("http server stopped", zap.Error(err))
	}
}

func startAndRegisterA2A(cfg *config.Config, rt *harness.Runtime, st *store.Store, log *zap.Logger) {
	if strings.TrimSpace(cfg.A2A.AuthToken) == "" {
		log.Warn("a2a enabled without auth_token; inbound endpoints are open")
	}
	a2aSrv, err := a2a.NewServer(cfg, rt, st)
	if err != nil {
		log.Fatal("a2a server", zap.Error(err))
	}
	a2aSrv.Start(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		a2a.RegisterToRegistries(ctx, cfg.A2A)
	})
}
