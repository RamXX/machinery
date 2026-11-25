// Command crm is the reference CRM service.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RamXX/machinery/examples/crm/internal/api"
	"github.com/RamXX/machinery/examples/crm/internal/store"
)

func main() {
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Fatalf("crm: %v", err)
	}
}

func run() error {
	addr := os.Getenv("CRM_ADDR")
	if addr == "" {
		addr = "127.0.0.1:18080"
	}
	appURL, adminURL := os.Getenv("CRM_DATABASE_URL"), os.Getenv("CRM_ADMIN_DATABASE_URL")
	if appURL == "" || adminURL == "" {
		return errors.New("CRM_DATABASE_URL and CRM_ADMIN_DATABASE_URL must be set")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	st, err := store.Open(appURL)
	if err != nil {
		return err
	}
	defer st.Close()
	handler, err := api.New(st, api.TablesDir())
	if err != nil {
		return err
	}
	go st.MigrateUntilDone(ctx, adminURL, handler.Lifecycles()...)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	log.Printf("crm: listening on %s", ln.Addr())
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shut); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	log.Print("crm: stopped")
	return nil
}
