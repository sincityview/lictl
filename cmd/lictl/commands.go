package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	libvirtclient "github.com/sincityview/lictl/internal/libvirt"
	"github.com/sincityview/lictl/internal/plan"
	"github.com/sincityview/lictl/internal/state"
	"github.com/sincityview/lictl/internal/config"
)

var autoApprove bool

// resolveProject определяет директорию проекта по имени из реестра или CWD
func resolveProject(args []string) (string, error) {
	if len(args) == 0 || args[0] == "" {
		return filepath.Abs(".")
	}

	name := args[0]
	registry, err := config.LoadRegistry()
	if err != nil {
		return "", fmt.Errorf("ошибка загрузки реестра: %w", err)
	}

	project := registry.FindProject(name)
	if project == nil {
		return "", fmt.Errorf("проект '%s' не найден. Используйте 'lictl list' для списка проектов", name)
	}

	return project.Path, nil
}

// resolveProjectWithExtra определяет проект и возвращает оставшиеся аргументы
// Используется для reboot: lictl reboot [project] <vm|all>
func resolveProjectWithExtra(args []string) (string, []string, error) {
	if len(args) == 0 {
		dir, err := filepath.Abs(".")
		return dir, args, err
	}

	// Проверяем первый аргумент — это имя проекта или имя VM?
	registry, err := config.LoadRegistry()
	if err != nil {
		return "", nil, fmt.Errorf("ошибка загрузки реестра: %w", err)
	}

	if project := registry.FindProject(args[0]); project != nil {
		// Первый аргумент — проект
		return project.Path, args[1:], nil
	}

	// Первый аргумент — VM, проект берём из CWD
	dir, _ := filepath.Abs(".")
	return dir, args, nil
}

func runInit(args []string) error {
	if _, err := os.Stat("lictl.yaml"); err == nil {
		return fmt.Errorf("lictl.yaml уже существует")
	}

	template := `# lictl.yaml — описание желаемого состояния
# Документация: https://github.com/sincityview/lictl

provider:
  libvirt:
    uri: "qemu:///system"

resources:
  base_images:
    - name: debian-13
      path: /var/lib/libvirt/images/debian-13-genericcloud-amd64.qcow2

  storage:
    - name: my-pool
      type: dir
      path: /var/lib/libvirt/my-storage

  networks:
    - name: my-net
      mode: nat
      subnet: 10.10.0.0/24
      dhcp:
        start: 10.10.0.100
        end: 10.10.0.200

  vms:
    - name: vm-1
      base_image: debian-13
      storage: my-pool
      cpu: 2
      memory: 2048
      networks:
        - my-net
      autostart: true
      cloud_init:
        hostname: vm-1
        network:
          dhcp: true
        users:
          - name: deploy
            ssh_authorized_keys:
              - ssh-ed25519 AAAA...
            sudo: true
            shell: /bin/bash
`
	if err := os.WriteFile("lictl.yaml", []byte(template), 0644); err != nil {
		return fmt.Errorf("ошибка создания lictl.yaml: %w", err)
	}

	fmt.Println("✓ Создан lictl.yaml")
	fmt.Println("  Отредактируй файл и запусти: lictl plan")

	// Регистрируем в реестре
	projectName := ""
	if len(args) > 0 && args[0] != "" {
		projectName = args[0]
	} else {
		cwd, _ := filepath.Abs(".")
		projectName = filepath.Base(cwd)
	}

	registry, err := config.LoadRegistry()
	if err != nil {
		fmt.Printf("  предупреждение: не удалось загрузить реестр: %v\n", err)
		return nil
	}

	cwd, _ := filepath.Abs(".")
	if err := registry.RegisterProject(projectName, cwd); err != nil {
		fmt.Printf("  предупреждение: не удалось зарегистрировать проект: %v\n", err)
	} else {
		fmt.Printf("  Проект '%s' зарегистрирован. Используй: lictl status %s\n", projectName, projectName)
	}

	return nil
}

func runPlan(dir string) error {
	cfgPath := filepath.Join(dir, "lictl.yaml")
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("ошибка валидации: %w", err)
	}

	if err := config.ValidateSubnets(cfg.Resources.Networks); err != nil {
		return fmt.Errorf("ошибка валидации подсетей: %w", err)
	}

	store := state.NewStore(dir)
	if err := store.Load(); err != nil {
		return fmt.Errorf("ошибка загрузки состояния: %w", err)
	}

	engine := plan.NewEngine(store)
	planResult, err := engine.Plan(cfg)
	if err != nil {
		return fmt.Errorf("ошибка генерации плана: %w", err)
	}

	plan.PrintPlan(planResult)

	return nil
}

func runApply(dir string) error {
	cfgPath := filepath.Join(dir, "lictl.yaml")
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("ошибка валидации: %w", err)
	}

	store := state.NewStore(dir)
	if err := store.Load(); err != nil {
		return fmt.Errorf("ошибка загрузки состояния: %w", err)
	}

	engine := plan.NewEngine(store)
	planResult, err := engine.Plan(cfg)
	if err != nil {
		return fmt.Errorf("ошибка генерации плана: %w", err)
	}

	plan.PrintPlan(planResult)

	if planResult.Summary.Total == 0 || planResult.Summary.Create+planResult.Summary.Update+planResult.Summary.Delete == 0 {
		fmt.Println("\nНет изменений для применения.")
		return nil
	}

	if !autoApprove {
		if !plan.ConfirmPlan(planResult) {
			fmt.Println("Отменено.")
			return nil
		}
	}

	conn := libvirtclient.NewConnection(cfg.Provider.Libvirt.URI)
	if err := conn.Connect(); err != nil {
		return fmt.Errorf("ошибка подключения к libvirt: %w", err)
	}
	defer conn.Disconnect()

	executor := plan.NewExecutor(conn, store, dir)
	result, err := executor.Execute(planResult, cfg)
	if err != nil {
		return fmt.Errorf("ошибка выполнения плана: %w", err)
	}

	plan.PrintResult(result)

	return nil
}

func runDestroy(dir string) error {
	cfgPath := filepath.Join(dir, "lictl.yaml")
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return err
	}

	store := state.NewStore(dir)
	if err := store.Load(); err != nil {
		return fmt.Errorf("ошибка загрузки состояния: %w", err)
	}

	allResources := store.GetAllResources()
	var toDelete []state.Resource
	for _, r := range allResources {
		if r.Owned {
			toDelete = append(toDelete, r)
		}
	}

	if len(toDelete) == 0 {
		fmt.Println("Нет ресурсов для удаления.")
		return nil
	}

	fmt.Println("Ресурсы для удаления (созданы lictl):")
	for _, r := range toDelete {
		fmt.Printf("  - %s (%s)\n", r.Name, r.Type)
	}

	ignored := len(allResources) - len(toDelete)
	if ignored > 0 {
		fmt.Printf("\n  (ещё %d ресурсов в state будут оставлены — не是我的)\n", ignored)
	}

	if !autoApprove {
		fmt.Println("\nУдалить только эти ресурсы? (да/нет)")
		fmt.Print("> ")
		var input string
		fmt.Scanln(&input)
		if input != "да" && input != "y" && input != "yes" {
			fmt.Println("Отменено.")
			return nil
		}
	}

	conn := libvirtclient.NewConnection(cfg.Provider.Libvirt.URI)
	if err := conn.Connect(); err != nil {
		return fmt.Errorf("ошибка подключения к libvirt: %w", err)
	}
	defer conn.Disconnect()

	domainManager := libvirtclient.NewDomainManager(conn)
	networkManager := libvirtclient.NewNetworkManager(conn)
	storageManager := libvirtclient.NewStorageManager(conn)

	for _, r := range toDelete {
		if r.Type == state.ResourceDomain {
			if err := domainManager.DeleteDomain(r.Name, true); err != nil {
				fmt.Printf("  ошибка удаления VM %s: %v\n", r.Name, err)
			} else {
				fmt.Printf("  ✓ VM %s удалена\n", r.Name)
			}
		}
	}

	for _, r := range toDelete {
		if r.Type == state.ResourceNetwork {
			if err := networkManager.DeleteNetwork(r.Name); err != nil {
				fmt.Printf("  ошибка удаления сети %s: %v\n", r.Name, err)
			} else {
				fmt.Printf("  ✓ Сеть %s удалена\n", r.Name)
			}
		}
	}

	for _, r := range toDelete {
		if r.Type == state.ResourceStorage {
			if err := storageManager.DeletePool(r.Name); err != nil {
				fmt.Printf("  ошибка удаления пула %s: %v\n", r.Name, err)
			} else {
				fmt.Printf("  ✓ Пул %s удалён\n", r.Name)
			}
		}
	}

	for _, r := range toDelete {
		store.RemoveResource(r.ID)
	}
	if err := store.Save(); err != nil {
		return fmt.Errorf("ошибка сохранения состояния: %w", err)
	}

	fmt.Printf("\nУдалено %d ресурсов. State обновлён.\n", len(toDelete))
	return nil
}

func runStatus(dir string, outputFormat string) error {
	cfgPath := filepath.Join(dir, "lictl.yaml")
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return err
	}

	store := state.NewStore(dir)
	if err := store.Load(); err != nil {
		return fmt.Errorf("ошибка загрузки состояния: %w", err)
	}

	resources := store.GetAllResources()
	if len(resources) == 0 {
		fmt.Println("Нет управляемых ресурсов.")
		return nil
	}

	conn := libvirtclient.NewConnection(cfg.Provider.Libvirt.URI)
	defer conn.Disconnect()

	domainManager := libvirtclient.NewDomainManager(conn)

	var statuses []plan.ResourceStatus
	for _, r := range resources {
		status := plan.ResourceStatus{
			Name:   r.Name,
			Type:   string(r.Type),
			Status: string(r.Status),
		}

		if r.Type == state.ResourceDomain {
			if r.IP != "" {
				status.IP = r.IP
			} else if ip, err := domainManager.GetDomainIP(r.Name); err == nil && ip != "" {
				status.IP = ip
				r.SetIP(ip)
			}
			if mac, err := domainManager.GetDomainMAC(r.Name); err == nil && mac != "" {
				status.MAC = mac
			}

			var drifts []string
			if info, err := domainManager.GetDomainInfo(r.Name); err == nil {
				liveCPU := int(info.VCPUs)
				liveMem := int(info.Memory / 1024)

				status.CPU = fmt.Sprintf("%d", liveCPU)
				status.Memory = fmt.Sprintf("%dMiB", liveMem)

				if r.ExpectedCPU > 0 && liveCPU != r.ExpectedCPU {
					drifts = append(drifts, fmt.Sprintf("CPU %d→%d", r.ExpectedCPU, liveCPU))
				}
				if r.ExpectedMemory > 0 && liveMem != r.ExpectedMemory {
					drifts = append(drifts, fmt.Sprintf("MEM %d→%dMiB", r.ExpectedMemory, liveMem))
				}
			}

			// IP drift через ARP
			if r.IP != "" {
				if actualIP, err := domainManager.GetDomainIPActual(r.Name); err == nil && actualIP != "" && actualIP != r.IP {
					drifts = append(drifts, fmt.Sprintf("IP %s→%s", r.IP, actualIP))
				}
			}

			if len(drifts) > 0 {
				status.Drift = strings.Join(drifts, ", ")
			}

			if disk, err := domainManager.GetDomainDiskSize(r.Name); err == nil && disk != "" {
				status.Disk = disk
			}
			store.UpdateResource(&r)
		}

		statuses = append(statuses, status)
	}

	store.Save()

	if outputFormat == "json" {
		data, err := json.MarshalIndent(statuses, "", "  ")
		if err != nil {
			return fmt.Errorf("ошибка сериализации JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}
	plan.PrintStatus(statuses)
	return nil
}

func runImport(dir string) error {
	cfgPath := filepath.Join(dir, "lictl.yaml")
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return err
	}

	store := state.NewStore(dir)
	if err := store.Load(); err != nil {
		return fmt.Errorf("ошибка загрузки состояния: %w", err)
	}

	conn := libvirtclient.NewConnection(cfg.Provider.Libvirt.URI)
	if err := conn.Connect(); err != nil {
		return fmt.Errorf("ошибка подключения к libvirt: %w", err)
	}
	defer conn.Disconnect()

	importer := plan.NewImporter(conn, store)
	result, err := importer.ImportAll(cfg)
	if err != nil {
		return fmt.Errorf("ошибка импорта: %w", err)
	}

	fmt.Println("Импорт завершён:")
	fmt.Printf("  Пулов хранения: %d\n", result.Storage)
	fmt.Printf("  Сетей: %d\n", result.Networks)
	fmt.Printf("  ВМ: %d\n", result.Domains)

	total := result.Storage + result.Networks + result.Domains
	fmt.Printf("\nВсего импортировано: %d ресурсов\n", total)

	return nil
}

func runValidate(dir string) error {
	cfgPath := filepath.Join(dir, "lictl.yaml")
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("ошибка валидации: %w", err)
	}

	if err := config.ValidateSubnets(cfg.Resources.Networks); err != nil {
		return fmt.Errorf("ошибка валидации подсетей: %w", err)
	}

	expandedVMs := config.ExpandAllVMs(cfg.Resources.VMs)

	fmt.Println("✓ YAML-файл валиден")
	fmt.Printf("  URI: %s\n", cfg.Provider.Libvirt.URI)
	fmt.Printf("  Пулов: %d, Сетей: %d, ВМ: %d\n",
		len(cfg.Resources.Storage),
		len(cfg.Resources.Networks),
		len(expandedVMs))
	return nil
}

func runCloudInitGenerate() error {
	fmt.Println("Генерация cloud-init ISO...")
	fmt.Println("⚠ Генерация cloud-init ещё не реализована")
	return nil
}

func runReboot(dir string, vmArgs []string) error {
	cfgPath := filepath.Join(dir, "lictl.yaml")
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return err
	}

	store := state.NewStore(dir)
	if err := store.Load(); err != nil {
		return fmt.Errorf("ошибка загрузки состояния: %w", err)
	}

	conn := libvirtclient.NewConnection(cfg.Provider.Libvirt.URI)
	if err := conn.Connect(); err != nil {
		return fmt.Errorf("ошибка подключения к libvirt: %w", err)
	}
	defer conn.Disconnect()

	domainManager := libvirtclient.NewDomainManager(conn)

	resources := store.GetAllResources()

	var toReboot []state.Resource
	if len(vmArgs) > 0 && vmArgs[0] == "all" {
		for _, r := range resources {
			if r.Type == state.ResourceDomain && r.Owned {
				toReboot = append(toReboot, r)
			}
		}
	} else if len(vmArgs) > 0 {
		for _, name := range vmArgs {
			found := false
			for _, r := range resources {
				if r.Type == state.ResourceDomain && r.Name == name {
					toReboot = append(toReboot, r)
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("  VM %s не найдена в state\n", name)
			}
		}
	} else {
		fmt.Println("Использование: lictl reboot <имя> | lictl reboot all")
		return nil
	}

	if len(toReboot) == 0 {
		fmt.Println("Нет VM для перезагрузки.")
		return nil
	}

	fmt.Println("Перезагрузка VM:")
	for _, r := range toReboot {
		fmt.Printf("  - %s... ", r.Name)
		if err := domainManager.RebootDomain(r.Name); err != nil {
			fmt.Printf("ошибка: %v\n", err)
		} else {
			fmt.Println("OK")
		}
	}

	fmt.Printf("\nПерезагружено %d VM. Подожди ~30 сек для получения IP.\n", len(toReboot))
	return nil
}

func runList() error {
	registry, err := config.LoadRegistry()
	if err != nil {
		return fmt.Errorf("ошибка загрузки реестра: %w", err)
	}

	if len(registry.Projects) == 0 {
		fmt.Println("Нет зарегистрированных проектов.")
		fmt.Println("  Используй: lictl init [имя] для создания проекта")
		return nil
	}

	fmt.Println()
	fmt.Printf("  %-20s %-40s\n", "NAME", "PATH")
	fmt.Println("  " + strings.Repeat("-", 60))

	for _, p := range registry.Projects {
		fmt.Printf("  %-20s %-40s\n", p.Name, p.Path)
	}

	fmt.Println()
	fmt.Printf("  Всего проектов: %d\n", len(registry.Projects))
	return nil
}
