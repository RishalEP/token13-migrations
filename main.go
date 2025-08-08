package main

import (
	"bootstrap/configure"
	"bootstrap/telemetry"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"quicknode/core"
	"quicknode/global"
	"quicknode/model/database"
	"quicknode/router"
	"quicknode/service"
	"syscall"
	"time"
)

func main() {
	var ctx = context.Background()
	var l = telemetry.NewLogger()
	var path = flag.String("c", "config_main.yml", "config path")

	flag.Parse()

	err := configure.InitApolloClient(l.Sugar())
	var conf *core.Config
	if err != nil {
		l.Warn("apollo client init fail")
		configure.MustInitViperLocalByPath(*path)
		conf = configure.ViperMustGetAll[core.Config]()
	} else {
		var apollo = configure.ReadEnvConfig[configure.Apollo]()
		conf = configure.MustGet[core.Config](apollo.NamespaceNames[0])
	}

	//shutdown := telemetry.Init(ctx, l, conf.Otel.Service, conf.Otel.Endpoints.GRPC, conf.Otel.Endpoints.HTTP)
	//defer shutdown()

	_, err = global.New(conf.ConfigPath)
	if err != nil {
		log.Fatalf("Failed to initialize global configuration: %v", err)
	}

	cleanup := global.InitGlobal()
	defer cleanup()
	db := global.MustGetDB(global.DBNameToken13)
	err = db.AutoMigrate(&database.WalletTransactionHistory{})
	if err != nil {
		log.Fatalf("AutoMigrate failed: %v", err)
	}
	log.Println("Connected to db")

	transactionService, err := service.NewTransactionService(
		conf.TronGrid.URL,
		conf.TronGrid.APIKey,
		conf.EtherScan.URL,
		conf.EtherScan.APIKey,
		conf.QuickNode.TronEndpoint,
		conf.QuickNode.EthEndpoint,
		conf.QuickNode.APIKey,
		conf.QuickNode.APIKeyHeader,
	)
	if err != nil {
		log.Fatalf("Failed to create transaction service: %v", err)
	}

	// Start HTTP server
	r := router.NewRouter(transactionService, conf.Server.APIKey)
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", conf.Server.Port),
		Handler: r.Engine(),
	}

	go func() {
		log.Printf("HTTP server listening on port %d", conf.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		log.Println("Shutting down HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("HTTP server shutdown failed: %v", err)
		}
		log.Println("HTTP server stopped")
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

}
