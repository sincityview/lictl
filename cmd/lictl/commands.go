package main

import (
	"fmt"
	"os"

	libvirtclient "github.com/alex/lictl/internal/libvirt"
	"github.com/alex/lictl/internal/plan"
	"github.com/alex/lictl/internal/state"
	"github.com/alex/lictl/internal/config"
)

var autoApprove bool

func runInit() error {
	if _, err := os.Stat("lictl.yaml"); err == nil {
		return fmt.Errorf("lictl.yaml уже существует")
	}

	template := `# lictl.yaml — описание желаемого состояния
provider:
  libvirt:
    uri: "qemu:///system"

resources:
  # Пулы хранения
  storage: []
  # Пример:
  # - name: default
  #   type: dir
  #   path: /var/lib/libvirt/images
  #   autostart: true

  # Сети
  networks: []
  # Пример:
  # - name: mgmt
  #   mode: nat
  #   subnet: 10.10.0.0/24
  #   dhcp:
  #     start: 10.10.0.100
  #     end: 10.10.0.200

  # Виртуальные машины
  vms: []
  # Пример:
  # - name: test-vm
  #   base_image: ubuntu-24.04-server-cloudimg-amd64.img
  #   storage_pool: default
  #   cpu: 2
  #   memory: 2048
  #   disk: 20Gi
  #   networks:
  #     - name: mgmt
  #   cloud_init:
  #     hostname: test-vm
  #     users:
  #       - name: ubuntu
  #         ssh_authorized_keys:
  #           - ssh-ed25519 AAAA...
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
	executor := plan.NewExecutor(conn, store, ".")
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

	// Получаем все ресурсы
	resources := store.GetAllResources()
	if len(resources) == 0 {
		fmt.Println("Нет управляемых ресурсов для удаления.")
		return nil
	}

	fmt.Println("Ресурсы для удаления:")
	for _, r := range resources {
		fmt.Printf("  - %s (%s)\n", r.Name, r.Type)
	}

	// Запрашиваем подтверждение
	if !autoApprove {
		fmt.Println("\nУдалить все эти ресурсы? (да/нет)")
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
	for _, r := range resources {
		if r.Type == state.ResourceDomain {
			if err := domainManager.DeleteDomain(r.Name, true); err != nil {
				fmt.Printf("  ошибка удаления VM %s: %v\n", r.Name, err)
			} else {
				fmt.Printf("  ✓ VM %s удалена\n", r.Name)
			}
		}
	}

	// Удаляем сети
	for _, r := range resources {
		if r.Type == state.ResourceNetwork {
			if err := networkManager.DeleteNetwork(r.Name); err != nil {
				fmt.Printf("  ошибка удаления сети %s: %v\n", r.Name, err)
			} else {
				fmt.Printf("  ✓ Сеть %s удалена\n", r.Name)
			}
		}
	}

	// Удаляем пулы
	for _, r := range resources {
		if r.Type == state.ResourceStorage {
			if err := storageManager.DeletePool(r.Name); err != nil {
				fmt.Printf("  ошибка удаления пула %s: %v\n", r.Name, err)
			} else {
				fmt.Printf("  ✓ Пул %s удалён\n", r.Name)
			}
		}
	}

	// Очищаем state
	store.Clear()
	if err := store.Save(); err != nil {
		return fmt.Errorf("ошибка сохранения состояния: %w", err)
	}

	fmt.Println("\nВсе ресурсы удалены.")
	return nil
}

func runStatus() error {
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
	var statuses []plan.ResourceStatus

	for _, r := range resources {
		status := plan.ResourceStatus{
			Name:   r.Name,
			Type:   string(r.Type),
			Status: string(r.Status),
			IP:     r.IP,
		}
		statuses = append(statuses, status)
	}

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
