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
- [x] Движок плана (diff + apply)
- [x] Команды plan, apply, destroy, status
- [x] Импорт существующих ресурсов
- [x] Тесты

## Приоритет 2 (Улучшения)

- [ ] Полноценная cloud-init ISO генерация (чистый Go)
- [ ] Клонирование base images
- [ ] Автоматическое определение IP после старта VM
- [ ] Обновление VM без пересоздания (hot-reload)
- [ ] Поддержка snapshot'ов
- [ ] Мульти-хост (управление VM на нескольких серверах)
- [ ] Шаблоны/переменные в YAML
- [ ] Вывод в JSON/TOML формат
- [ ] Цветной вывод (fatih/color)

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

1. SSH поддержка требует настройки ключей вручную
2. Cloud-init ISO генерируется через genisoimage (требуется установка)
3. Нет поддержки encrypted storage
4. Нет rollback при ошибке apply
5. State хранится в JSON (не зашифрован)
