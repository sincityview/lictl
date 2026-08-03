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
- [x] Команды plan, apply, destroy, status
- [x] Импорт существующих ресурсов
- [x] Тесты

## Приоритет 2 (Улучшения)

- [ ] Автоматическое определение IP после старта VM
- [ ] Клонирование base images (qemu-img resize)
- [ ] Обновление VM без пересоздания (hot-reload)
- [ ] Поддержка snapshot'ов
- [ ] Мульти-хост (управление VM на нескольких серверах)
- [ ] Шаблоны/переменные в YAML
- [ ] Вывод в JSON/TOML формат
- [ ] Цветной вывод (fatih/color)
- [ ] State для существующих ресурсов (авто-импорт при apply)
- [ ] Расширение disk (qemu-img resize при изменении размера)

## Приоритет 3 (Продвинутое)

- [ ] GUI (веб-интерфейс)
- [ ] Интеграция с Prometheus/metrics
- [ ] CI/CD пайплайны (GitHub Actions)
- [ ] Terraform provider (обратная совместимость)
- [ ] Ansible инвентарь из state
- [ ] Автоматическое резервное копирование
- [ ] Rollback при ошибках
- [ ] Параллельное создание VM
- [ ] Ограничение ресурсов (CPU/MEM quota)
- [ ] Интеграция с Terraform state backend

## Идеи

- [ ] Плагины для разных гипервизоров (Xen, VMware, VirtualBox)
- [ ] Webhook'и при изменениях
- [ ] Chat-бот для управления VM
- [ ] Автоматическое масштабирование (HPA-like)
- [ ] Cost tracking (виртуальные ресурсы → стоимость)

## Известные ограничения

1. Cloud-init ISO генерируется через genisoimage (требуется установка + sudo)
2. Storage pool и network не попадают в state если уже существуют в libvirt (нужен import)
3. Нет поддержки encrypted storage
4. Нет rollback при ошибке apply
5. State хранится в JSON (не зашифрован)
6. Нет поддержки disk size (образ используется как есть)
