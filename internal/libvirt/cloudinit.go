package libvirt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sincityview/lictl/internal/config"
)

// runCommand выполняет shell-команду
func runCommand(cmd string) error {
	execCmd := exec.Command("sh", "-c", cmd)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	return execCmd.Run()
}

// runCommandQuiet выполняет shell-команду без вывода stderr
func runCommandQuiet(cmd string) error {
	execCmd := exec.Command("sh", "-c", cmd)
	return execCmd.Run()
}

// copyFile копирует файл
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}

// CloudInitGenerator генерирует cloud-init файлы
type CloudInitGenerator struct {
	outputDir string
}

// NewCloudInitGenerator создаёт генератор cloud-init
func NewCloudInitGenerator(outputDir string) *CloudInitGenerator {
	return &CloudInitGenerator{outputDir: outputDir}
}

// GenerateFiles генерирует meta-data и user-data файлы
func (g *CloudInitGenerator) GenerateFiles(vm config.VMConfig) (*CloudInitFiles, error) {
	if vm.CloudInit == nil {
		return nil, nil
	}

	// Создаём директорию для файлов
	filesDir := filepath.Join(g.outputDir, vm.Name+"-cloud-init")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		return nil, fmt.Errorf("ошибка создания директории: %w", err)
	}

	// Генерируем meta-data
	metaData := g.generateMetaData(vm)
	metaDataPath := filepath.Join(filesDir, "meta-data")
	if err := os.WriteFile(metaDataPath, []byte(metaData), 0644); err != nil {
		return nil, fmt.Errorf("ошибка записи meta-data: %w", err)
	}

	// Генерируем user-data
	userData := g.generateUserData(vm)
	userDataPath := filepath.Join(filesDir, "user-data")
	if err := os.WriteFile(userDataPath, []byte(userData), 0644); err != nil {
		return nil, fmt.Errorf("ошибка записи user-data: %w", err)
	}

	return &CloudInitFiles{
		MetaDataPath: metaDataPath,
		UserDataPath: userDataPath,
		Directory:    filesDir,
	}, nil
}

// CloudInitFiles пути к сгенерированным файлам
type CloudInitFiles struct {
	MetaDataPath string
	UserDataPath string
	Directory    string
}

// generateMetaData генерирует meta-data
func (g *CloudInitGenerator) generateMetaData(vm config.VMConfig) string {
	var sb strings.Builder

	hostname := vm.Name
	if vm.CloudInit != nil && vm.CloudInit.Hostname != "" {
		hostname = vm.CloudInit.Hostname
	}

	sb.WriteString(fmt.Sprintf("instance-id: %s\n", vm.Name))
	sb.WriteString(fmt.Sprintf("local-hostname: %s\n", hostname))

	return sb.String()
}

// generateUserData генерирует user-data в формате cloud-config
func (g *CloudInitGenerator) generateUserData(vm config.VMConfig) string {
	if vm.CloudInit == nil {
		return "#cloud-config\n"
	}

	var sb strings.Builder

	sb.WriteString("#cloud-config\n\n")

	// Пользователи
	if len(vm.CloudInit.Users) > 0 {
		sb.WriteString("users:\n")
		for _, user := range vm.CloudInit.Users {
			sb.WriteString(fmt.Sprintf("  - name: %s\n", user.Name))

			if len(user.SSHPublicKeys) > 0 {
				sb.WriteString("    ssh_authorized_keys:\n")
				for _, key := range user.SSHPublicKeys {
					sb.WriteString(fmt.Sprintf("      - %s\n", key))
				}
			}

			if user.Sudo {
				sb.WriteString("    sudo: ['ALL=(ALL) NOPASSWD:ALL']\n")
			}

			if user.Shell != "" {
				sb.WriteString(fmt.Sprintf("    shell: %s\n", user.Shell))
			}

			if user.LockPassword {
				sb.WriteString("    lock_passwd: true\n")
			}
		}
		sb.WriteString("\n")
	}

	// SSH
	sb.WriteString("ssh_pwauth: true\n\n")

	// Пароль для пользователей (если указан)
	hasPassword := false
	for _, user := range vm.CloudInit.Users {
		if user.Password != "" {
			hasPassword = true
			break
		}
	}
	if hasPassword {
		sb.WriteString("chpasswd:\n")
		sb.WriteString("  list: |\n")
		for _, user := range vm.CloudInit.Users {
			if user.Password != "" {
				sb.WriteString(fmt.Sprintf("    %s:%s\n", user.Name, user.Password))
			}
		}
		sb.WriteString("  expire: false\n")
		sb.WriteString("\n")
	}

	// bootcmd — удаляет конфиг base image ДО настройки сети
	sb.WriteString("bootcmd:\n")
	sb.WriteString("  - rm -f /etc/netplan/00-installer-config.yaml\n")
	sb.WriteString("  - rm -f /etc/netplan/00-installer-config-kvm.yaml\n")
	sb.WriteString("  - rm -f /etc/netplan/50-cloud-init.yaml\n")
	sb.WriteString("\n")

	// Network — write_files с netplan
	if vm.CloudInit.Network != nil {
		net := vm.CloudInit.Network
		sb.WriteString("write_files:\n")
		sb.WriteString("  - path: /etc/netplan/99-lictl.yaml\n")
		sb.WriteString("    content: |\n")
		sb.WriteString("      network:\n")
		sb.WriteString("        version: 2\n")
		sb.WriteString("        ethernets:\n")
		sb.WriteString("          enp1s0:\n")
		if net.IP != "" {
			prefix := net.IPPrefix
			if prefix == 0 {
				prefix = 24
			}
			sb.WriteString("            addresses:\n")
			sb.WriteString(fmt.Sprintf("              - %s/%d\n", net.IP, prefix))
			if net.Gateway != "" {
				sb.WriteString(fmt.Sprintf("            routes:\n"))
				sb.WriteString(fmt.Sprintf("              - to: default\n"))
				sb.WriteString(fmt.Sprintf("                via: %s\n", net.Gateway))
			}
			if len(net.DNS) > 0 {
				sb.WriteString("            nameservers:\n")
				sb.WriteString("              addresses:\n")
				for _, dns := range net.DNS {
					sb.WriteString(fmt.Sprintf("                - %s\n", dns))
				}
			}
		} else {
			if net.DHCP {
				sb.WriteString("            dhcp4: true\n")
			}
		}
		sb.WriteString("\n")
	}

	// Рост корневого раздела
	sb.WriteString("growpart:\n")
	sb.WriteString("  mode: auto\n")
	sb.WriteString("  devices: ['/']\n\n")

	// Пакеты
	if len(vm.CloudInit.Packages) > 0 {
		sb.WriteString("package_update: true\n")
		sb.WriteString("package_upgrade: true\n\n")
		sb.WriteString("packages:\n")
		for _, pkg := range vm.CloudInit.Packages {
			sb.WriteString(fmt.Sprintf("  - %s\n", pkg))
		}
		sb.WriteString("\n")
	}

	// Команды — netplan apply первый, потом пользовательские
	if vm.CloudInit.Network != nil || len(vm.CloudInit.RunCmd) > 0 {
		sb.WriteString("runcmd:\n")
		if vm.CloudInit.Network != nil {
			sb.WriteString("  - netplan apply\n")
		}
		for _, cmd := range vm.CloudInit.RunCmd {
			sb.WriteString(fmt.Sprintf("  - %s\n", cmd))
		}
		sb.WriteString("\n")
	}

	// Финал
	sb.WriteString("final_message: cloud-init completed in $UPTIME seconds\n")

	return sb.String()
}

// GenerateISO генерирует cloud-init ISO
func (g *CloudInitGenerator) GenerateISO(vm config.VMConfig, isoPath string) (string, error) {
	files, err := g.GenerateFiles(vm)
	if err != nil {
		return "", err
	}

	if files == nil {
		return "", nil
	}

	// Если isoPath не указан, генерируем по умолчанию
	if isoPath == "" {
		isoPath = filepath.Join(g.outputDir, vm.Name+"-cloud-init.iso")
	}

	// Генерируем ISO с помощью genisoimage через sudo
	cmd := fmt.Sprintf("sudo genisoimage -output %s -volid cidata -joliet -rock %s/meta-data %s/user-data",
		isoPath, files.Directory, files.Directory)

	// Выполняем команду
	if err := runCommandQuiet(cmd); err != nil {
		return "", fmt.Errorf("ошибка генерации ISO: %w", err)
	}

	return isoPath, nil
}

// GetISOPath возвращает путь к ISO для VM
func (g *CloudInitGenerator) GetISOPath(vmName string) string {
	return filepath.Join(g.outputDir, vmName+"-cloud-init.iso")
}

// ValidateCloudInit проверяет валидность cloud-init конфигурации
func ValidateCloudInit(ci *config.CloudInit) error {
	if ci == nil {
		return nil
	}

	if ci.Hostname == "" {
		return fmt.Errorf("hostname обязателен")
	}

	for _, user := range ci.Users {
		if user.Name == "" {
			return fmt.Errorf("имя пользователя обязательно")
		}
	}

	return nil
}
