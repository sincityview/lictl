package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	cfgFile string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "lictl",
		Short: "Легковесный IaC-инструмент для управления VM через libvirt",
		Long: `lictl — декларативный CLI для управления виртуальными машинами.
Описываешь желаемое состояние в YAML, lictl apply доводит реальность до него.`,
	}

	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(planCmd())
	rootCmd.AddCommand(applyCmd())
	rootCmd.AddCommand(destroyCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(importCmd())
	rootCmd.AddCommand(validateCmd())
	rootCmd.AddCommand(cloudInitCmd())
	rootCmd.AddCommand(rebootCmd())
	rootCmd.AddCommand(completionCmd())
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Инициализация проекта, создание lictl.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit()
		},
	}
}

func planCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plan",
		Short: "Показать что изменится при применении",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlan()
		},
	}
}

func applyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Применить изменения для достижения желаемого состояния",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApply()
		},
	}
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Пропустить подтверждение")
	return cmd
}

func destroyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Удалить все управляемые ресурсы",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDestroy()
		},
	}
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Пропустить подтверждение")
	return cmd
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Показать текущее состояние управляемых ресурсов",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus()
		},
	}
}

func importCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import",
		Short: "Импорт существующих ресурсов в state",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImport()
		},
	}
}

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Валидация YAML-файла плана",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate()
		},
	}
}

func cloudInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud-init",
		Short: "Управление cloud-init",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "generate",
		Short: "Генерация cloud-init ISO из плана",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCloudInitGenerate()
		},
	})

	return cmd
}

func rebootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reboot [имя | all]",
		Short: "Перезагрузить управляемые VM",
		Long:  "Перезагружает VM для обновления DHCP lease.\n  lictl reboot <имя> — перезагрузить конкретную VM\n  lictl reboot all — перезагрузить все owned VM",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReboot(args)
		},
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Показать версию lictl",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("lictl %s (commit: %s, built: %s)\n", version, commit, date)
		},
	}
}
