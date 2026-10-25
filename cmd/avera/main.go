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
	"github.com/pirakansa/avera/internal/collect/accounts"
	"github.com/pirakansa/avera/internal/collect/network"
	"github.com/pirakansa/avera/internal/collect/storage"
	"github.com/pirakansa/avera/internal/collect/sudo"
	"github.com/pirakansa/avera/internal/collect/usb"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	localSource := sudo.LocalSource{Root: "/"}
	collectors := []app.Registration{
		{Name: "network", Collector: network.WithForwarding{Sockets: network.Collector{IncludeServices: true}}},
		{Name: "accounts", Collector: accounts.Collector{Source: localSource}},
		{Name: "sudo", Collector: sudo.Collector{Source: localSource}},
		{Name: "storage", Collector: storage.Collector{}},
		{Name: "usb", Collector: usb.Collector{IncludeInput: true}},
	}
	cmd := root.New(app.Service{EffectiveUID: os.Geteuid, Collectors: collectors})
	err := cmd.ExecuteContext(ctx)
	if err != nil && shared.Code(err) != 2 {
		fmt.Fprintln(os.Stderr, "avera:", err)
	}
	os.Exit(shared.Code(err))
}
