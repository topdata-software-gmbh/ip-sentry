package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	appcfg "github.com/topdata/topdata-ip-aggregator/internal/config"
	"github.com/topdata/topdata-ip-aggregator/internal/monitor"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start monitoring logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		var cfg appcfg.Config
		if err := viper.Unmarshal(&cfg); err != nil {
			return err
		}

		m, err := monitor.New(cfg)
		if err != nil {
			return err
		}
		defer m.Close()

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		return m.Run(ctx)
	},
}
