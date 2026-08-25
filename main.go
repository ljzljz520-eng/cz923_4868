package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pharmacycounter/catalog"
	"pharmacycounter/config"
	"pharmacycounter/httpapi"
	"pharmacycounter/query"
	"pharmacycounter/queue"
	"pharmacycounter/service"
	"pharmacycounter/store"
)

func main() {
	configuration, err := config.FromEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	storage, err := store.Open(configuration.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer storage.Close()
	business, err := service.New(storage, queue.DefaultPolicy())
	if err != nil {
		log.Fatal(err)
	}
	if err := business.SeedCounters(configuration.Counters); err != nil {
		log.Fatal(err)
	}
	logger := log.New(os.Stdout, "pharmacy-counter ", log.LstdFlags)
	api := httpapi.New(business, query.New(storage), catalog.Default(), configuration.StaticPath, logger)
	server := &http.Server{
		Addr:              configuration.Address,
		Handler:           api.Handler(),
		ReadHeaderTimeout: time.Duration(configuration.ReadTimeoutSeconds) * time.Second,
		WriteTimeout:      time.Duration(configuration.WriteTimeoutSeconds) * time.Second,
	}
	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopped
		server.Close()
	}()
	logger.Printf("药房取药台启动于 %s", configuration.Address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
