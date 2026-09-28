// Command guppy is a terminal UI for Google Tasks.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"guppy/internal/auth"
	"guppy/internal/config"
	"guppy/internal/gtasks"
	"guppy/internal/tui"
)

func main() {
	reauth := flag.Bool("reauth", false, "discard the saved sign-in token and sign in again")
	flag.Parse()

	if err := run(*reauth); err != nil {
		fmt.Fprintln(os.Stderr, "guppy:", err)
		os.Exit(1)
	}
}

func run(reauth bool) error {
	if err := config.EnsureDir(); err != nil {
		return fmt.Errorf("preparing config directory: %w", err)
	}

	creds, err := auth.ResolveCredentials()
	if err != nil {
		return err
	}

	if reauth {
		_ = os.Remove(config.TokenPath())
	}

	ctx := context.Background()
	httpClient, err := auth.Client(ctx, creds, config.TokenPath())
	if err != nil {
		return fmt.Errorf("sign-in failed: %w", err)
	}

	svc, err := gtasks.New(ctx, httpClient)
	if err != nil {
		return err
	}

	return tui.Run(svc)
}
