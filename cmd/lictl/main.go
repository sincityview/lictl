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
	rootCmd.AddCommand(listCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [имя_проекта]",
		Short: "Инициализация проекта, создание lictl.yaml",
		Long:  "Создаёт lictl.yaml в текущей директории и регистрирует проект.\n  lictl init — имя берётся из имени директории\n  lictl init myproject — задать имя вручную",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(args)
		},
	}
}

func planCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plan [проект]",
		Short: "Показать что изменится при применении",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := resolveProject(args)
			if err != nil {
				return err
			}
			return runPlan(dir)
		},
	}
}

func applyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply [проект]",
		Short: "Применить изменения для достижения желаемого состояния",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := resolveProject(args)
			if err != nil {
				return err
			}
			return runApply(dir)
		},
	}
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Пропустить подтверждение")
	return cmd
}

func destroyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "destroy [проект]",
		Short: "Удалить все управляемые ресурсы",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := resolveProject(args)
			if err != nil {
				return err
			}
			return runDestroy(dir)
		},
	}
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Пропустить подтверждение")
	return cmd
}

func statusCmd() *cobra.Command {
	var outputFormat string
	cmd := &cobra.Command{
		Use:   "status [проект]",
		Short: "Показать текущее состояние управляемых ресурсов",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := resolveProject(args)
			if err != nil {
				return err
			}
			return runStatus(dir, outputFormat)
		},
	}
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Формат вывода: table, json")
	return cmd
}

func importCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import [проект]",
		Short: "Импорт существующих ресурсов в state",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := resolveProject(args)
			if err != nil {
				return err
			}
			return runImport(dir)
		},
	}
}

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate [проект]",
		Short: "Валидация YAML-файла плана",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := resolveProject(args)
			if err != nil {
				return err
			}
			return runValidate(dir)
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
		Use:   "reboot [проект] <имя|all>",
		Short: "Перезагрузить управляемые VM",
		Long:  "Перезагружает VM для обновления DHCP lease.\n  lictl reboot <имя> — перезагрузить конкретную VM\n  lictl reboot all — перезагрузить все owned VM\n  lictl reboot myproject all — указать проект",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, vmArgs, err := resolveProjectWithExtra(args)
			if err != nil {
				return err
			}
			return runReboot(dir, vmArgs)
		},
	}
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Показать зарегистрированные проекты",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList()
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
