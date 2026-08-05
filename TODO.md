# TODO — Планы по развитию lictl

## Приоритет 1 (MVP)

- [x] CLI фреймворк (cobra)
- [x] YAML схема и валидация
- [x] Расширение диапазонов (worker-{1..3})
- [x] Хранение состояния (JSON)
- [x] XML генераторы (Domain, Network, Storage)
- [x] CRUD для Storage pools
- [x] CRUD для Networks
- [x] CRUD для Domains (VM)
- [x] Cloud-init генерация (meta-data, user-data)
- [x] Cloud-init ISO генерация (через genisoimage)
- [x] Движок плана (diff + apply)
- [x] Команды plan, apply, destroy, status, reboot
- [x] Импорт существующих ресурсов
- [x] Тесты
- [x] Owned flag в state — destroy не удаляет чужие ресурсы
- [x] Overlay qcow2 для sharing base images
- [x] base_images секция в YAML (imя + path)
- [x] Network config в cloud-init (DHCP/static)
- [x] Очистка base image netplan из overlay (virt-customize)
- [x] Configurable password в cloud-init (опционально)
- [x] Status показывает CPU/memory/IP (docker compose ps стиль)
- [x] Команда reboot (per-VM или all)

## Приоритет 2 (Улучшения)

- [ ] base_images: загрузка по URL (скачивание если файла нет)
- [ ] Клонирование base images (qemu-img resize)
- [ ] Обновление VM без пересоздания (hot-reload)
- [ ] Поддержка snapshot'ов
- [ ] Мульти-хост (управление VM на нескольких серверах)
- [ ] Шаблоны/переменные в YAML
- [ ] Вывод в JSON/TOML формат
- [ ] Цветной вывод (fatih/color)
- [ ] Расширение disk (qemu-img resize при изменении размера)
- [ ] Rollback при ошибках apply
- [ ] Параллельное создание VM

## Приоритет 3 (Продвинутое)

- [ ] GUI (веб-интерфейс)
- [ ] Интеграция с Prometheus/metrics
- [ ] CI/CD пайплайны (GitHub Actions)
- [ ] Terraform provider (обратная совместимость)
- [ ] Ansible инвентарь из state
- [ ] Автоматическое резервное копирование
- [ ] Ограничение ресурсов (CPU/MEM quota)

## Идеи

- [ ] Плагины для разных гипервизоров (Xen, VMware, VirtualBox)
- [ ] Webhook'и при изменениях
- [ ] Автоматическое масштабирование (HPA-like)
- [ ] Cost tracking (виртуальные ресурсы → стоимость)

## Известные ограничения

1. Cloud-init ISO генерируется через genisoimage (требуется установка + sudo)
2. virt-customize замедляет apply (~3-4 сек на VM)
3. Нет поддержки encrypted storage
4. State хранится в JSON (не зашифрован)
5. Нет поддержки disk size (образ используется как есть)
6. base_images url не реализован (только path)
