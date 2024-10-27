package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"golang.org/x/sync/errgroup"
)

const (
	ollamaArg  = "serve"
	ollamaEnv  = "OLLAMA_HOST=127.0.0.1:11434"
	ollamaName = "ollama.exe"

	routineNum = -1
)

var (
	cmd *exec.Cmd
)

func main() {
	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(routineNum)

	g.Go(func() error {
		if err := start(ctx); err != nil {
			fmt.Printf("%s\n", err.Error())
			os.Exit(1)
		}
		return nil
	})

	s := make(chan os.Signal, 1)

	// kill (no param) default send syscanll.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can"t be caught, so don't need add it
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)

	g.Go(func() error {
		<-s
		return stop()
	})

	if err := g.Wait(); err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}

	os.Exit(0)
}

// nolint:gosec
func start(ctx context.Context) error {
	ex, err := os.Executable()
	if err != nil {
		return err
	}

	path := filepath.Dir(ex)

	cmd = exec.CommandContext(ctx, filepath.Join(path, ollamaName), ollamaArg)
	cmd.Env = append(cmd.Environ(), ollamaEnv)

	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}

func stop() error {
	var err error

	if cmd != nil {
		err = cmd.Process.Kill()
	}

	return err
}
