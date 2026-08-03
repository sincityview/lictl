# lictl — Легковесный IaC для Libvirt

Декларативный CLI-инструмент для управления виртуальными машинами через libvirt. Описываешь желаемое состояние в YAML — `lictl apply` доводит реальность до него.

## Почему не Terraform?

- **Быстрее** — прямой вызов libvirt API, без абстракций Terraform
- **Проще** — один YAML-файл вместо HCL + state + провайдер
- **Нативнее** — заточен под libvirt, не пытается быть universal
- **Легче** — нет зависимостей от Terraform/OpenTofu

## Установка

```bash
# Из исходников
git clone https://github.com/alex/lictl.git
cd lictl
go build -o lictl ./cmd/lictl/

# Или установить глобально
go install github.com/alex/lictl/cmd/lictl@latest
```

## Быстрый старт

```bash
# 1. Инициализация проекта
mkdir my-cluster && cd my-cluster
lictl init

# 2. Отредактируй lictl.yaml
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

## Формат YAML (`lictl.yaml`)

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
      subnet: 10.10.0.0/24
      dhcp:
        start: 10.10.0.100
        end: 10.10.0.200
      autostart: true

  # Виртуальные машины
  vms:
    - name: control-plane-1
      base_image: ubuntu-24.04-server-cloudimg-amd64.img
      storage_pool: default
      cpu: 2
      memory: 4096
      disk: 40Gi
      networks:
        - name: mgmt
          ip: 10.10.0.10
      cloud_init:
        hostname: cp-1
        users:
          - name: ubuntu
            ssh_authorized_keys:
              - ssh-ed25519 AAAA...
            sudo: true
        packages:
          - qemu-guest-agent
        runcmd:
          - systemctl enable qemu-guest-agent
      autostart: true

    # Расширение диапазонов
    - name: worker-{1..3}
      base_image: ubuntu-24.04-server-cloudimg-amd64.img
      storage_pool: default
      cpu: 4
      memory: 8192
      disk: 100Gi
      networks:
        - name: mgmt
      cloud_init:
        hostname: worker-{N}
        users:
          - name: ubuntu
            ssh_authorized_keys:
              - ssh-ed25519 AAAA...
      autostart: true
```

## Команды

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

### Опции

| Флаг | Команда | Описание |
|------|---------|----------|
| `--auto-approve` | `apply`, `destroy` | Пропустить подтверждение |

## Примеры

### Создание кластера из 4 VM

```yaml
# lictl.yaml
provider:
  libvirt:
    uri: "qemu:///system"

resources:
  storage:
    - name: vms
      type: dir
      path: /var/lib/libvirt/images/vms
      autostart: true

  networks:
    - name: mgmt
      mode: nat
      subnet: 10.10.0.0/24
      dhcp:
        start: 10.10.0.100
        end: 10.10.0.200

  vms:
    - name: cp-{1..3}
      base_image: ubuntu-24.04-server-cloudimg-amd64.img
      storage_pool: vms
      cpu: 2
      memory: 4096
      disk: 40Gi
      cloud_init:
        hostname: cp-{N}
        users:
          - name: ubuntu
            ssh_authorized_keys:
              - ssh-ed25519 AAAA...
      autostart: true

    - name: worker-{1..3}
      base_image: ubuntu-24.04-server-cloudimg-amd64.img
      storage_pool: vms
      cpu: 4
      memory: 8192
      disk: 100Gi
      cloud_init:
        hostname: worker-{N}
        users:
          - name: ubuntu
            ssh_authorized_keys:
              - ssh-ed25519 AAAA...
      autostart: true
```

```bash
lictl plan
# Вывод:
#   + создать пул vms (dir)
#   + создать сеть mgmt (nat)
#   + создать VM cp-1 (CPU: 2, RAM: 4096MiB)
#   + создать VM cp-2 (CPU: 2, RAM: 4096MiB)
#   + создать VM cp-3 (CPU: 2, RAM: 4096MiB)
#   + создать VM worker-1 (CPU: 4, RAM: 8192MiB)
#   + создать VM worker-2 (CPU: 4, RAM: 8192MiB)
#   + создать VM worker-3 (CPU: 4, RAM: 8192MiB)
# Итого: 8 создать, 0 обновить, 0 удалить

lictl apply --auto-approve
# Создание пула vms... готово
# Создание сети mgmt... готово
# Создание VM cp-1... готово
# ...

lictl status
# ИМЯ              СТАТУС    IP              CPU   ПАМЯТЬ
# cp-1              working   10.10.0.10      2     4096
# cp-2              working   10.10.0.11      2     4096
# ...
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

## Архитектура

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

## Зависимости

- Go 1.21+
- libvirt (для удалённых хостов через SSH)
- genisoimage (опционально, для генерации cloud-init ISO)

## Лицензия

MIT
