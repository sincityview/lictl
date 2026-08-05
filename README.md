# lictl — Легковесный IaC для Libvirt

Декларативный CLI для управления VM через libvirt. Описываешь желаемое состояние в YAML — `lictl apply` доводит реальность до него.

## Зависимости

- Go 1.26+
- libvirt (`qemu:///system` или удалённый через SSH)
- genisoimage (для cloud-init ISO)
- libguestfs-tools (для очистки base image)

## Сборка

```bash
# Зависимости
sudo apt install -y genisoimage libguestfs-tools

# Из исходников
git clone https://github.com/sincityview/lictl.git
cd lictl
make build
```

## Команды

| Команда | Описание |
|---------|----------|
| `lictl init` | Создать `lictl.yaml` из шаблона |
| `lictl validate` | Проверить YAML |
| `lictl plan` | Показать что изменится |
| `lictl apply` | Применить изменения |
| `lictl destroy` | Удалить управляемые ресурсы |
| `lictl status` | Статус ресурсов (CPU, RAM, IP) |
| `lictl reboot <name\|all>` | Перезагрузить VM |
| `lictl import` | Импорт существующих ресурсов |
| `lictl version` | Версия |

Флаги: `--auto-approve` для `apply` и `destroy`.

## Autocompletion

```bash
# Bash
eval "$(lictl completion bash)"

# Zsh
eval "$(lictl completion zsh)"

# Fish
lictl completion fish > ~/.config/fish/completions/lictl.fish
```

## Пример

```yaml
vars:
  ssh_key: ssh-ed25519 AAAA... user@host

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
          static_ip: 10.10.0.10/24
          gateway: 10.10.0.1
          dns:
            - 8.8.8.8
        packages:
          - nginx
          - curl
        runcmd:
          - systemctl enable nginx
          - systemctl start nginx
        users:
          - name: deploy
            ssh_authorized_keys:
              - ${ssh_key}
            sudo: true
            shell: /bin/bash
```
