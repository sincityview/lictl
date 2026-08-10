package plan

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sincityview/lictl/internal/config"
	libvirtclient "github.com/sincityview/lictl/internal/libvirt"
	"github.com/sincityview/lictl/internal/state"
	"github.com/sincityview/lictl/internal/xml"
)

// Executor выполняет план
type Executor struct {
	conn     *libvirtclient.Connection
	store    *state.Store
	basePath string // Базовый путь для хранения образов
}

// NewExecutor создаёт исполнитель плана
func NewExecutor(conn *libvirtclient.Connection, store *state.Store, basePath string) *Executor {
	return &Executor{
		conn:     conn,
		store:    store,
		basePath: basePath,
	}
}

// Execute выполняет план
func (e *Executor) Execute(plan *Plan, cfg *config.Config) (*Result, error) {
	result := &Result{}

	// Сортируем изменения в порядке зависимостей
	sortedChanges := SortChanges(plan.Changes)

	for _, change := range sortedChanges {
		var err error

		switch change.Type {
		case Create:
			err = e.executeCreate(change, cfg)
		case Update:
			err = e.executeUpdate(change, cfg)
		case Delete:
			err = e.executeDelete(change)
		case NoOp:
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("ошибка выполнения для %s: %w", change.Name, err)
		}

		result.Applied = append(result.Applied, AppliedChange{
			Change: change,
			Status: "success",
		})
	}

	result.Summary = Summary{
		Create: len(result.Applied),
		Total:  len(result.Applied),
	}

	return result, nil
}

// Result результат выполнения
type Result struct {
	Applied []AppliedChange
	Summary Summary
}

// AppliedChange применённое изменение
type AppliedChange struct {
	Change Change
	Status string
	Error  error
}

// executeCreate выполняет создание ресурса
func (e *Executor) executeCreate(change Change, cfg *config.Config) error {
	switch change.ResourceType {
	case state.ResourceStorage:
		return e.createStorage(change, cfg)
	case state.ResourceNetwork:
		return e.createNetwork(change, cfg)
	case state.ResourceDomain:
		return e.createDomain(change, cfg)
	}
	return fmt.Errorf("неизвестный тип ресурса: %s", change.ResourceType)
}

// executeUpdate выполняет обновление ресурса
func (e *Executor) executeUpdate(change Change, cfg *config.Config) error {
	// Для обновления пересоздаём ресурс
	if err := e.executeDelete(change); err != nil {
		return err
	}
	return e.executeCreate(change, cfg)
}

// executeDelete выполняет удаление ресурса
func (e *Executor) executeDelete(change Change) error {
	switch change.ResourceType {
	case state.ResourceStorage:
		return e.deleteStorage(change)
	case state.ResourceNetwork:
		return e.deleteNetwork(change)
	case state.ResourceDomain:
		return e.deleteDomain(change)
	}
	return fmt.Errorf("неизвестный тип ресурса: %s", change.ResourceType)
}

// createStorage создаёт пул хранения
func (e *Executor) createStorage(change Change, cfg *config.Config) error {
	storageCfg, ok := change.Desired.(config.StorageConfig)
	if !ok {
		return fmt.Errorf("невалидная конфигурация storage")
	}

	manager := libvirtclient.NewStorageManager(e.conn)

	// Проверяем существует ли уже
	if manager.PoolExists(storageCfg.Name) {
		// Импортируем в state как не nuestro
		if e.store.GetResourceByName(storageCfg.Name, state.ResourceStorage) == nil {
			fmt.Printf("  Пул %s уже существует — импортирую в state (не是我的)\n", storageCfg.Name)
			resource := state.NewResource(storageCfg.Name, storageCfg.Name, state.ResourceStorage)
			resource.UpdateStatus(state.StatusRunning)
			resource.Owned = false
			resource.SetConfigHash(state.HashConfig(storageCfg))
			e.store.AddResource(resource)
			return e.store.Save()
		}
		return nil
	}

	// Генерируем XML для пула
	xmlCfg := &xml.StoragePoolConfig{
		Name:      storageCfg.Name,
		Type:      storageCfg.Type,
		Path:      storageCfg.Path,
		VGName:    storageCfg.VgName,
		Autostart: storageCfg.Autostart,
	}

	_, err := manager.CreatePool(xmlCfg)
	if err != nil {
		return err
	}

	// Сохраняем в state
	resource := state.NewResource(storageCfg.Name, storageCfg.Name, state.ResourceStorage)
	resource.UpdateStatus(state.StatusRunning)
	resource.Owned = true
	resource.SetConfigHash(state.HashConfig(storageCfg))
	e.store.AddResource(resource)

	return e.store.Save()
}

// deleteStorage удаляет пул хранения
func (e *Executor) deleteStorage(change Change) error {
	manager := libvirtclient.NewStorageManager(e.conn)

	if !manager.PoolExists(change.Name) {
		return nil
	}

	err := manager.DeletePool(change.Name)
	if err != nil {
		return err
	}

	// Удаляем из state
	e.store.RemoveResource(change.Name)
	return e.store.Save()
}

// createNetwork создаёт сеть
func (e *Executor) createNetwork(change Change, cfg *config.Config) error {
	networkCfg, ok := change.Desired.(config.NetworkConfig)
	if !ok {
		return fmt.Errorf("невалидная конфигурация network")
	}

	manager := libvirtclient.NewNetworkManager(e.conn)

	// Проверяем существует ли уже
	if manager.NetworkExists(networkCfg.Name) {
		// Импортируем в state как не nuestro
		if e.store.GetResourceByName(networkCfg.Name, state.ResourceNetwork) == nil {
			fmt.Printf("  Сеть %s уже существует — импортирую в state (не是我的)\n", networkCfg.Name)
			resource := state.NewResource(networkCfg.Name, networkCfg.Name, state.ResourceNetwork)
			resource.UpdateStatus(state.StatusRunning)
			resource.Owned = false
			resource.SetConfigHash(state.HashConfig(networkCfg))
			e.store.AddResource(resource)
			return e.store.Save()
		}
		return nil
	}

	// Генерируем XML для сети
	xmlCfg := &xml.NetworkConfig{
		Name:      networkCfg.Name,
		Bridge:    networkCfg.Bridge,
		Mode:      networkCfg.Mode,
		Subnet:    networkCfg.Subnet,
		Autostart: networkCfg.Autostart,
	}

	if networkCfg.DHCP != nil {
		xmlCfg.DHCP = &xml.DHCPConfig{
			RangeStart: networkCfg.DHCP.Start,
			RangeEnd:   networkCfg.DHCP.End,
		}
	}

	_, err := manager.CreateNetwork(xmlCfg)
	if err != nil {
		return err
	}

	// Сохраняем в state
	resource := state.NewResource(networkCfg.Name, networkCfg.Name, state.ResourceNetwork)
	resource.UpdateStatus(state.StatusRunning)
	resource.Owned = true
	resource.SetConfigHash(state.HashConfig(networkCfg))
	e.store.AddResource(resource)

	return e.store.Save()
}

// deleteNetwork удаляет сеть
func (e *Executor) deleteNetwork(change Change) error {
	manager := libvirtclient.NewNetworkManager(e.conn)

	if !manager.NetworkExists(change.Name) {
		return nil
	}

	err := manager.DeleteNetwork(change.Name)
	if err != nil {
		return err
	}

	e.store.RemoveResource(change.Name)
	return e.store.Save()
}

// createDomain создаёт VM
func (e *Executor) createDomain(change Change, cfg *config.Config) error {
	vmCfg, ok := change.Desired.(config.VMConfig)
	if !ok {
		return fmt.Errorf("невалидная конфигурация VM")
	}

	domainManager := libvirtclient.NewDomainManager(e.conn)

	// Проверяем существует ли уже
	if domainManager.DomainExists(vmCfg.Name) {
		existing := e.store.GetResourceByName(vmCfg.Name, state.ResourceDomain)
		if existing != nil {
			// Уже в state — ничего не делаем
			return nil
		}
		// В libvirt но нет в state — импортируем
		fmt.Printf("  VM %s уже существует — импортирую в state (не是我的)\n", vmCfg.Name)
		resource := state.NewResource(vmCfg.Name, vmCfg.Name, state.ResourceDomain)
		resource.UpdateStatus(state.StatusRunning)
		resource.Owned = false
		resource.SetConfigHash(state.HashConfig(vmCfg))
		e.store.AddResource(resource)
		return e.store.Save()
	}

	// Определяем путь к образу
	var storagePath string
	poolDir := e.basePath
	if vmCfg.StoragePool != "" {
		for _, s := range cfg.Resources.Storage {
			if s.Name == vmCfg.StoragePool {
				poolDir = s.Path
				break
			}
		}
	}

	var baseImagePath string
	if vmCfg.BaseImage != "" {
		// Пробуем path из base_images
		var err error
		baseImagePath, err = cfg.ResolveBaseImage(vmCfg.BaseImage)
		if err != nil {
			// Если path не указан — пробуем url
			if bi := cfg.FindBaseImage(vmCfg.BaseImage); bi != nil && bi.URL != "" {
				baseImagePath, err = downloadBaseImage(bi.Name, bi.URL, poolDir)
			}
			if err != nil {
				return err
			}
		}

		// Если base_image относительный — резолвим
		if !filepath.IsAbs(baseImagePath) {
			baseImagePath = filepath.Join(poolDir, baseImagePath)
		}

		// Overlay (рабочий образ VM) кладём в storage pool
		storagePath = filepath.Join(poolDir, vmCfg.Name+".qcow2")
		if err := createOverlay(baseImagePath, storagePath); err != nil {
			return fmt.Errorf("ошибка создания overlay для %s: %w", vmCfg.Name, err)
		}

		// Удаляем netplan конфиг base image из overlay чтобы не конфликтовал с cloud-init
		if err := cleanOverlayNetplan(storagePath); err != nil {
			fmt.Printf("  предупреждение: не удалось очистить netplan в overlay: %v\n", err)
		}
	} else {
		storagePath = filepath.Join(e.basePath, vmCfg.Name+".qcow2")
	}

	fmt.Printf("  создание VM %s... ", vmCfg.Name)

	// Генерируем cloud-init
	var cloudInitISO string
	if vmCfg.CloudInit != nil {
		// Кладём ISO в директорию хранения чтобы qemu имел доступ
		isoDir := filepath.Dir(storagePath)
		isoPath := filepath.Join(isoDir, vmCfg.Name+"-cloud-init.iso")

		// Генерируем временные файлы в basePath, затем создаём ISO в storageDir
		ciGenerator := libvirtclient.NewCloudInitGenerator(e.basePath)
		generatedISO, err := ciGenerator.GenerateISO(vmCfg, isoPath)
		if err != nil {
			return fmt.Errorf("ошибка генерации cloud-init: %w", err)
		}
		cloudInitISO = generatedISO
	}

	// Создаём VM
	result, err := domainManager.CreateDomain(vmCfg, storagePath, cloudInitISO)
	if err != nil {
		fmt.Println("ошибка")
		return err
	}
	fmt.Println("OK")

	// Сохраняем в state
	resource := state.NewResource(vmCfg.Name, vmCfg.Name, state.ResourceDomain)
	resource.UpdateStatus(state.StatusRunning)
	resource.Owned = true
	resource.SetLibvirtID(result.UUID)
	resource.SetConfigHash(state.HashConfig(vmCfg))
	resource.ExpectedCPU = vmCfg.CPU
	resource.ExpectedMemory = vmCfg.Memory

	// Сохраняем сконфигурированный IP из cloud-init
	if vmCfg.CloudInit != nil && vmCfg.CloudInit.Network != nil && vmCfg.CloudInit.Network.IP != "" {
		resource.SetIP(vmCfg.CloudInit.Network.IP)
	}

	e.store.AddResource(resource)

	return e.store.Save()
}

// deleteDomain удаляет VM
func (e *Executor) deleteDomain(change Change) error {
	domainManager := libvirtclient.NewDomainManager(e.conn)

	if !domainManager.DomainExists(change.Name) {
		return nil
	}

	err := domainManager.DeleteDomain(change.Name, true)
	if err != nil {
		return err
	}

	e.store.RemoveResource(change.Name)
	return e.store.Save()
}

// createOverlay создаёт qcow2 overlay поверх базового образа
func createOverlay(baseImage, overlayPath string) error {
	cmd := exec.Command("sudo", "qemu-img", "create", "-f", "qcow2", "-b", baseImage, "-F", "qcow2", overlayPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("qemu-img: %s: %w", string(output), err)
	}
	return nil
}

// cleanOverlayNetplan удаляет netplan конфиг и кэш cloud-init из overlay
func cleanOverlayNetplan(overlayPath string) error {
	cmd := exec.Command("sudo", "virt-customize", "-a", overlayPath,
		"--delete", "/etc/netplan/50-cloud-init.yaml",
		"--delete", "/etc/netplan/00-installer-config.yaml",
		"--delete", "/etc/netplan/00-installer-config-kvm.yaml",
		"--delete", "/var/lib/cloud/instance",
		"--delete", "/var/lib/cloud/data",
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("virt-customize: %w", err)
	}
	return nil
}

// downloadBaseImage скачивает base image по URL в директорию pool
func downloadBaseImage(name, url, destDir string) (string, error) {
	ext := ".qcow2"
	if idx := strings.LastIndex(url, "."); idx != -1 {
		candidate := url[idx:]
		if len(candidate) <= 6 {
			ext = candidate
		}
	}

	destPath := filepath.Join(destDir, name+ext)

	// Если уже скачан — используем
	if fi, err := os.Stat(destPath); err == nil && fi.Size() > 0 {
		fmt.Printf("  base_image %s: уже загружен (%dMB)\n", name, fi.Size()/(1024*1024))
		return destPath, nil
	}

	fmt.Printf("  base_image %s: скачиваю %s...\n", name, url)

	// Скачиваем во временный файл (у пользователя нет прав на pool dir)
	tmpFile, err := os.CreateTemp("", "lictl-download-*"+ext)
	if err != nil {
		return "", fmt.Errorf("ошибка создания временного файла: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath) // чистим временный файл
	}()

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("ошибка скачивания %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ошибка скачивания %s: HTTP %d", url, resp.StatusCode)
	}

	written, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка записи %s: %w", tmpPath, err)
	}
	tmpFile.Close()

	fmt.Printf("  base_image %s: скачано %dMB\n", name, written/(1024*1024))

	// Копируем в pool dir через sudo
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("ошибка создания директории: %w", err)
	}

	cmd := exec.Command("sudo", "cp", tmpPath, destPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ошибка копирования в %s: %s: %w", destPath, string(output), err)
	}

	fmt.Printf("  base_image %s: установлено в %s\n", name, destPath)
	return destPath, nil
}
