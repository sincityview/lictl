package libvirt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alex/lictl/internal/config"
)

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

	// Команды
	if len(vm.CloudInit.RunCmd) > 0 {
		sb.WriteString("runcmd:\n")
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
// Примечание: требует genisoimage или mkisofs в системе
func (g *CloudInitGenerator) GenerateISO(vm config.VMConfig, isoPath string) (string, error) {
	files, err := g.GenerateFiles(vm)
	if err != nil {
		return "", err
	}

	if files == nil {
		return "", nil
	}

	// Проверяем наличие genisoimage
	if _, err := os.Stat("/usr/bin/genisoimage"); os.IsNotExist(err) {
		// Пробуем mkisofs
		if _, err := os.Stat("/usr/bin/mkisofs"); os.IsNotExist(err) {
			return files.Directory, fmt.Errorf("genisoimage или mkisofs не найдены")
		}
	}

	// Генерируем ISO
	// Примечание: в реальном проекте здесь будет вызов genisoimage
	// Пока возвращаем путь к директории с файлами
	return files.Directory, nil
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
