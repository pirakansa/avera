package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pirakansa/avera/internal/app"
	"github.com/pirakansa/avera/internal/cli/commands/root"
	"github.com/pirakansa/avera/internal/cli/shared"
	"github.com/pirakansa/avera/internal/collect/network"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cmd := root.New(app.Service{EffectiveUID: os.Geteuid, Collectors: []app.Registration{{Name: "network", Collector: network.WithForwarding{Sockets: network.Collector{IncludeServices: true}}}}})
	err := cmd.ExecuteContext(ctx)
	if err != nil && shared.Code(err) != 2 {
		fmt.Fprintln(os.Stderr, "avera:", err)
	}
	os.Exit(shared.Code(err))
}
