package cmd

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/metal-toolbox/bomservice/internal/app"
	"github.com/metal-toolbox/bomservice/internal/metrics"
	"github.com/metal-toolbox/bomservice/internal/model"
	"github.com/metal-toolbox/bomservice/internal/server"
	"github.com/metal-toolbox/bomservice/internal/store"
	"github.com/spf13/cobra"
)

var (
	shutdownTimeout = 10 * time.Second
)

// install server command
var cmdServer = &cobra.Command{
	Use:   "server",
	Short: "Run bomservice server",
	Run: func(cmd *cobra.Command, args []string) {
		app, termCh, err := app.New(model.AppKindServer, cfgFile, model.LogLevel(logLevel))
		if err != nil {
			log.Fatal(err)
		}

		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()
		repository, err := store.NewStore(ctx, app.Config, app.Logger)
		if err != nil {
			app.Logger.Fatal(err)
		}

		options := []server.Option{
			server.WithLogger(app.Logger),
			server.WithListenAddress(app.Config.ListenAddress),
			server.WithStore(repository),
			server.WithAuthMiddlewareConfig(app.Config.APIServerJWTAuth),
		}

		srv := server.New(options...)
		go func() {
			if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
				app.Logger.Fatal(err)
			}
		}()
		metrics.ListenAndServe()

		// sit around for term signal
		<-termCh
		app.Logger.Info("got TERM signal, shutting down server...")

		sCtx, sCancel := context.WithTimeout(ctx, shutdownTimeout)
		defer sCancel()
		if err := srv.Shutdown(sCtx); err != nil {
			app.Logger.Fatal("server shutdown error:", err)
		}
	},
}

// install command flags
func init() {
	rootCmd.AddCommand(cmdServer)
}
