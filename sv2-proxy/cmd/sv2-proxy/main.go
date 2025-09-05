package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	sv2proxy "github.com/ethereum-optimism/optimism/sv2-proxy"
)

func main() {
	var (
		userListenAddr string
		userListenPort int
		authListenAddr string
		authListenPort int
		elUserUpstream string
		elAuthUpstream string
	)

	flag.StringVar(&userListenAddr, "user-listen-addr", "0.0.0.0", "address to bind the user RPC proxy")
	flag.IntVar(&userListenPort, "user-listen-port", 8545, "port to bind the user RPC proxy")
	flag.StringVar(&authListenAddr, "auth-listen-addr", "0.0.0.0", "address to bind the auth RPC proxy")
	flag.IntVar(&authListenPort, "auth-listen-port", 8551, "port to bind the auth RPC proxy")
	flag.StringVar(&elUserUpstream, "el-user-upstream", "http://el:8545", "upstream EL user RPC (http(s)://host:port or ws(s)://host:port)")
	flag.StringVar(&elAuthUpstream, "el-auth-upstream", "http://el:8551", "upstream EL auth RPC (http(s)://host:port or ws(s)://host:port)")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start composite proxy with fixed ports
	user, userURL, err := sv2proxy.StartELUserProxy(ctx, userListenAddr, userListenPort, elUserUpstream)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start user proxy: %v\n", err)
		os.Exit(1)
	}
	defer user.Close(ctx)

	auth, authURL, err := sv2proxy.StartELAuthProxy(ctx, authListenAddr, authListenPort, elAuthUpstream)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start auth proxy: %v\n", err)
		os.Exit(1)
	}
	defer auth.Close(ctx)

	fmt.Printf("sv2-proxy started\nuser: %s -> %s\nauth: %s -> %s\n", userURL, elUserUpstream, authURL, elAuthUpstream)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	// Give proxies a moment to shutdown cleanly
	shutdownCtx, cancelShutdown := context.WithTimeout(ctx, 2*time.Second)
	defer cancelShutdown()
	_ = user.Close(shutdownCtx)
	_ = auth.Close(shutdownCtx)
}
