## lictl — Легковесный IaC для Libvirt

Декларативный CLI-инструмент для управления виртуальными машинами через libvirt. Описываешь желаемое состояние в YAML — `lictl apply` доводит реальность до него.

### Установка

```bash
# Зависимости
sudo apt install -y genisoimage

# Из исходников
git clone https://github.com/alex/lictl.git
cd lictl
GONOSUMCHECK=* GONOSUMDB=* GOINSECURE=* GOPROXY=direct go build -o lictl ./cmd/lictl/

# Или установить глобально
GONOSUMCHECK=* GONOSUMDB=* GOINSECURE=* GOPROXY=direct go install github.com/sincityview/lictl/cmd/lictl@latest
```

### Быстрый старт

```bash
# 1. Создай директорию проекта
mkdir my-cluster && cd my-cluster

# 2. Создай lictl.yaml (см. примеры ниже)
vim lictl.yaml

# 3. Посмотри что изменится
lictl plan

# 4. Примени изменения
lictl apply

# 5. Проверь статус
lictl status

# 6. Удали всё
lictl destroy
```

### Формат YAML (`lictl.yaml`)

```yaml
provider:
  libvirt:
    uri: "qemu:///system"           # или qemu+ssh://root@host/system

resources:
  # Пулы хранения
  storage:
    - name: default
      type: dir
      path: /var/lib/libvirt/images
      autostart: true

  # Виртуальные сети
  networks:
    - name: mgmt
      mode: nat
      bridge: virbr1                 # опционально
      subnet: 10.10.0.0/24
      dhcp:
        start: 10.10.0.100
        end: 10.10.0.200
      autostart: true

  # Виртуальные машины
  vms:
    - name: control-plane-1
      base_image: /var/lib/libvirt/images/debian-13-cloud.qcow2
      storage: default               # имя пула хранения
      cpu: 2
      memory: 4096
      cloud_init:
        hostname: cp-1
        users:
          - name: deploy
            ssh_authorized_keys:
              - ssh-ed25519 AAAA...
            sudo: true
            shell: /bin/bash
        packages:
          - qemu-guest-agent
        runcmd:
          - systemctl enable qemu-guest-agent
      autostart: true

    # Расширение диапазонов
    - name: worker-{1..3}
      base_image: /var/lib/libvirt/images/debian-13-cloud.qcow2
      storage: default
      cpu: 4
      memory: 8192
      cloud_init:
        hostname: worker-{N}
        users:
          - name: deploy
            ssh_authorized_keys:
              - ssh-ed25519 AAAA...
      autostart: true
```

### Команды

| Команда | Описание |
|---------|----------|
| `lictl init` | Инициализация проекта, создание `lictl.yaml` |
| `lictl validate` | Валидация YAML-файла |
| `lictl plan` | Показать что изменится при применении |
| `lictl apply` | Применить изменения для достижения желаемого состояния |
| `lictl destroy` | Удалить все управляемые ресурсы |
| `lictl status` | Показать текущее состояние ресурсов |
| `lictl import` | Импорт существующих ресурсов из libvirt |
| `lictl cloud-init generate` | Генерация cloud-init файлов |

#### Опции

| Флаг | Команда | Описание |
|------|---------|----------|
| `--auto-approve` | `apply`, `destroy` | Пропустить подтверждение |

## Примеры

### Создание VM с cloud-init

```bash
# 1. Скачай cloud-образ
wget https://cloud.debian.org/images/cloud/trixie/latest/debian-13-genericcloud-amd64.qcow2
sudo mv debian-13-genericcloud-amd64.qcow2 /var/lib/libvirt/images/

# 2. Создай проект
mkdir test && cd test

# 3. Создай lictl.yaml
cat > lictl.yaml << 'EOF'
provider:
  libvirt:
    uri: qemu:///system

resources:
  storage:
    - name: test-pool
      type: dir
      path: /var/lib/libvirt/test-storage

  networks:
    - name: test-net
      mode: nat
      bridge: virbr1
      subnet: 192.168.100.0/24
      dhcp:
        start: 192.168.100.2
        end: 192.168.100.254

  vms:
    - name: debian-test
      memory: 1024
      cpu: 2
      storage: test-pool
      base_image: /var/lib/libvirt/images/debian-13-genericcloud-amd64.qcow2
      cloud_init:
        hostname: debian-test
        users:
          - name: alex
            ssh_authorized_keys:
              - ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA... user@host
            sudo: true
            shell: /bin/bash
EOF

# 4. Примени
lictl apply

# 5. Проверь SSH
ssh deploy@<VM_IP>
```

### Удалённое управление (SSH)

```yaml
provider:
  libvirt:
    uri: "qemu+ssh://root@192.168.1.100/system"
```

### Импорт существующих ресурсов

```bash
# Импортируй все VM/сети/пулы из libvirt в state
lictl import

# Теперь они под управлением lictl
lictl status
```

### Архитектура

```
lictl/
├── cmd/lictl/           # CLI точка входа
├── internal/
│   ├── config/          # YAML схема и валидация
│   ├── libvirt/         # Обёртка над libvirt API
│   ├── state/           # Хранение состояния (JSON)
│   ├── plan/            # Движок плана (diff + apply)
│   └── xml/             # Генераторы XML
└── examples/            # Примеры YAML-планов
```

### Зависимости

- Go 1.26+
- libvirt (qemu:///system или удалённый через SSH)
- genisoimage (для генерации cloud-init ISO)

