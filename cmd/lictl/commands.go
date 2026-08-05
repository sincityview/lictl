package main

import (
	"fmt"
	"os"
	"path/filepath"

	libvirtclient "github.com/sincityview/lictl/internal/libvirt"
	"github.com/sincityview/lictl/internal/plan"
	"github.com/sincityview/lictl/internal/state"
	"github.com/sincityview/lictl/internal/config"
)

var autoApprove bool

func runInit() error {
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
          dhcp4: true
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
	return nil
}

func runPlan() error {
	cfg, err := config.LoadConfig("lictl.yaml")
	if err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("ошибка валидации: %w", err)
	}

	if err := config.ValidateSubnets(cfg.Resources.Networks); err != nil {
		return fmt.Errorf("ошибка валидации подсетей: %w", err)
	}

	// Загружаем state
	store := state.NewStore(".")
	if err := store.Load(); err != nil {
		return fmt.Errorf("ошибка загрузки состояния: %w", err)
	}

	// Генерируем план
	engine := plan.NewEngine(store)
	planResult, err := engine.Plan(cfg)
	if err != nil {
		return fmt.Errorf("ошибка генерации плана: %w", err)
	}

	// Выводим план
	plan.PrintPlan(planResult)

	return nil
}

func runApply() error {
	cfg, err := config.LoadConfig("lictl.yaml")
	if err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("ошибка валидации: %w", err)
	}

	// Загружаем state
	store := state.NewStore(".")
	if err := store.Load(); err != nil {
		return fmt.Errorf("ошибка загрузки состояния: %w", err)
	}

	// Генерируем план
	engine := plan.NewEngine(store)
	planResult, err := engine.Plan(cfg)
	if err != nil {
		return fmt.Errorf("ошибка генерации плана: %w", err)
	}

	// Выводим план
	plan.PrintPlan(planResult)

	// Если нет изменений, выходим
	if planResult.Summary.Total == 0 || planResult.Summary.Create+planResult.Summary.Update+planResult.Summary.Delete == 0 {
		fmt.Println("\nНет изменений для применения.")
		return nil
	}

	// Запрашиваем подтверждение
	if !autoApprove {
		if !plan.ConfirmPlan(planResult) {
			fmt.Println("Отменено.")
			return nil
		}
	}

	// Подключаемся к libvirt
	conn := libvirtclient.NewConnection(cfg.Provider.Libvirt.URI)
	if err := conn.Connect(); err != nil {
		return fmt.Errorf("ошибка подключения к libvirt: %w", err)
	}
	defer conn.Disconnect()

	// Выполняем план
	basePath, _ := filepath.Abs(".")
	executor := plan.NewExecutor(conn, store, basePath)
	result, err := executor.Execute(planResult, cfg)
	if err != nil {
		return fmt.Errorf("ошибка выполнения плана: %w", err)
	}

	// Выводим результат
	plan.PrintResult(result)

	return nil
}

func runDestroy() error {
	cfg, err := config.LoadConfig("lictl.yaml")
	if err != nil {
		return err
	}

	// Загружаем state
	store := state.NewStore(".")
	if err := store.Load(); err != nil {
		return fmt.Errorf("ошибка загрузки состояния: %w", err)
	}

	// Берём только ресурсы которые создали мы (owned=true)
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

	// Показываем что останется в state
	ignored := len(allResources) - len(toDelete)
	if ignored > 0 {
		fmt.Printf("\n  (ещё %d ресурсов в state будут оставлены — не是我的)\n", ignored)
	}

	// Запрашиваем подтверждение
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

	// Подключаемся к libvirt
	conn := libvirtclient.NewConnection(cfg.Provider.Libvirt.URI)
	if err := conn.Connect(); err != nil {
		return fmt.Errorf("ошибка подключения к libvirt: %w", err)
	}
	defer conn.Disconnect()

	// Удаляем ресурсы в обратном порядке (VM → Network → Storage)
	domainManager := libvirtclient.NewDomainManager(conn)
	networkManager := libvirtclient.NewNetworkManager(conn)
	storageManager := libvirtclient.NewStorageManager(conn)

	// Удаляем VM
	for _, r := range toDelete {
		if r.Type == state.ResourceDomain {
			if err := domainManager.DeleteDomain(r.Name, true); err != nil {
				fmt.Printf("  ошибка удаления VM %s: %v\n", r.Name, err)
			} else {
				fmt.Printf("  ✓ VM %s удалена\n", r.Name)
			}
		}
	}

	// Удаляем сети
	for _, r := range toDelete {
		if r.Type == state.ResourceNetwork {
			if err := networkManager.DeleteNetwork(r.Name); err != nil {
				fmt.Printf("  ошибка удаления сети %s: %v\n", r.Name, err)
			} else {
				fmt.Printf("  ✓ Сеть %s удалена\n", r.Name)
			}
		}
	}

	// Удаляем пулы
	for _, r := range toDelete {
		if r.Type == state.ResourceStorage {
			if err := storageManager.DeletePool(r.Name); err != nil {
				fmt.Printf("  ошибка удаления пула %s: %v\n", r.Name, err)
			} else {
				fmt.Printf("  ✓ Пул %s удалён\n", r.Name)
			}
		}
	}

	// Удаляем из state только удалённые ресурсы
	for _, r := range toDelete {
		store.RemoveResource(r.ID)
	}
	if err := store.Save(); err != nil {
		return fmt.Errorf("ошибка сохранения состояния: %w", err)
	}

	fmt.Printf("\nУдалено %d ресурсов. State обновлён.\n", len(toDelete))
	return nil
}

func runStatus() error {
	cfg, err := config.LoadConfig("lictl.yaml")
	if err != nil {
		return err
	}

	// Загружаем state
	store := state.NewStore(".")
	if err := store.Load(); err != nil {
		return fmt.Errorf("ошибка загрузки состояния: %w", err)
	}

	resources := store.GetAllResources()
	if len(resources) == 0 {
		fmt.Println("Нет управляемых ресурсов.")
		return nil
	}

	// Подключаемся к libvirt для получения актуальной информации
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

		// Для VM получаем дополнительную информацию
		if r.Type == state.ResourceDomain {
			if ip, err := domainManager.GetDomainIP(r.Name); err == nil && ip != "" {
				status.IP = ip
				r.SetIP(ip)
			}
			if mac, err := domainManager.GetDomainMAC(r.Name); err == nil && mac != "" {
				status.MAC = mac
			}
			if info, err := domainManager.GetDomainInfo(r.Name); err == nil {
				status.CPU = fmt.Sprintf("%d", info.VCPUs)
				status.Memory = fmt.Sprintf("%dMiB", info.Memory/1024)
			}
			if disk, err := domainManager.GetDomainDiskSize(r.Name); err == nil && disk != "" {
				status.Disk = disk
			}
			store.UpdateResource(&r)
		}

		statuses = append(statuses, status)
	}

	store.Save()
	plan.PrintStatus(statuses)
	return nil
}

func runImport() error {
	cfg, err := config.LoadConfig("lictl.yaml")
	if err != nil {
		return err
	}

	// Загружаем state
	store := state.NewStore(".")
	if err := store.Load(); err != nil {
		return fmt.Errorf("ошибка загрузки состояния: %w", err)
	}

	// Подключаемся к libvirt
	conn := libvirtclient.NewConnection(cfg.Provider.Libvirt.URI)
	if err := conn.Connect(); err != nil {
		return fmt.Errorf("ошибка подключения к libvirt: %w", err)
	}
	defer conn.Disconnect()

	// Импортируем ресурсы
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

func runValidate() error {
	cfg, err := config.LoadConfig("lictl.yaml")
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
	// TODO: Реализовать в Задаче 8
	fmt.Println("⚠ Генерация cloud-init ещё не реализована")
	return nil
}

func runReboot(args []string) error {
	cfg, err := config.LoadConfig("lictl.yaml")
	if err != nil {
		return err
	}

	store := state.NewStore(".")
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

	// Определяем список VM для перезагрузки
	var toReboot []state.Resource
	if len(args) > 0 && args[0] == "all" {
		// Все owned VM
		for _, r := range resources {
			if r.Type == state.ResourceDomain && r.Owned {
				toReboot = append(toReboot, r)
			}
		}
	} else if len(args) > 0 {
		// Конкретные VM по имени
		for _, name := range args {
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
