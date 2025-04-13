package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	dockerclient "github.com/docker/docker/client"
	"golang.org/x/sync/errgroup"

	"github.com/skpr/local-router/internal/certificates"
	"github.com/skpr/local-router/internal/docker"
	skprhandler "github.com/skpr/local-router/internal/handler"
)

func main() {
	var (
		cliLabel     = os.Getenv("SKPR_LOCAL_ROUTER_LABEL")
		cliAddrHTTP  = os.Getenv("SKPR_LOCAL_ROUTER_ADDR_HTTP")
		cliAddrHTTPS = os.Getenv("SKPR_LOCAL_ROUTER_ADDR_HTTPS")
	)

	cli, err := dockerclient.NewClientWithOpts(dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	certificateManager, err := certificates.NewManager("/mnt/certificates")
	if err != nil {
		panic(err)
	}

	handler := skprhandler.New()

	eg := new(errgroup.Group)

	// Handling for HTTPS (Automatic certificate provisioning).
	eg.Go(func() error {
		server := &http.Server{
			Addr: cliAddrHTTPS,
			TLSConfig: &tls.Config{
				GetCertificate: certificateManager.GetCertificate,
			},
			Handler: http.HandlerFunc(handler.Handle),
		}

		log.Println("Starting server: https")

		return server.ListenAndServeTLS("", "")
	})

	// Handling for HTTP.
	eg.Go(func() error {
		server := &http.Server{
			Addr:    cliAddrHTTP,
			Handler: http.HandlerFunc(handler.Handle),
		}

		log.Println("Starting server: http")

		return server.ListenAndServe()
	})

	// Task for syncing Docker containers to handler routes.
	eg.Go(func() error {
		for {
			routes, err := docker.GetRoutes(cli, cliLabel)
			if err != nil {
				return fmt.Errorf("failed to get routers: %w", err)
			}

			err = handler.SetRoutes(routes)
			if err != nil {
				return fmt.Errorf("failed to set routes: %w", err)
			}

			time.Sleep(10)
		}
	})

	err = eg.Wait()
	if err != nil {
		panic(err)
	}
}
