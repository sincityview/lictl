package plan

import (
	"fmt"
	"strings"
)

// PrintPlan выводит план в консоль
func PrintPlan(plan *Plan) {
	if len(plan.Changes) == 0 {
		fmt.Println("Нет изменений для применения.")
		return
	}

	fmt.Println("План изменений:")
	fmt.Println(strings.Repeat("=", 60))

	for _, change := range plan.Changes {
		symbol := changeSymbol(change.Type)
		fmt.Printf("  %s %s\n", symbol, change.Details)
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Итого: %d создать, %d обновить, %d удалить, %d без изменений\n",
		plan.Summary.Create,
		plan.Summary.Update,
		plan.Summary.Delete,
		plan.Summary.NoOp)
	fmt.Printf("Всего ресурсов: %d\n", plan.Summary.Total)
}

// PrintResult выводит результат выполнения
func PrintResult(result *Result) {
	if len(result.Applied) == 0 {
		fmt.Println("Нет применённых изменений.")
		return
	}

	fmt.Println("Результат применения:")
	fmt.Println(strings.Repeat("=", 60))

	for _, applied := range result.Applied {
		symbol := "✓"
		if applied.Error != nil {
			symbol = "✗"
		}
		fmt.Printf("  %s %s\n", symbol, applied.Change.Details)
		if applied.Error != nil {
			fmt.Printf("    Ошибка: %v\n", applied.Error)
		}
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Применено изменений: %d\n", len(result.Applied))
}

// PrintStatus выводит статус ресурсов
func PrintStatus(resources []ResourceStatus) {
	if len(resources) == 0 {
		fmt.Println("Нет управляемых ресурсов.")
		return
	}

	fmt.Println()
	fmt.Printf("  %-20s %-10s %-12s %-20s %-18s %-10s %-10s\n",
		"NAME", "STATUS", "CPU", "MEMORY", "IP", "DISK", "DRIFT")
	fmt.Println("  " + strings.Repeat("-", 100))

	for _, r := range resources {
		ip := r.IP
		if ip == "" {
			ip = "-"
		}
		disk := r.Disk
		if disk == "" {
			disk = "-"
		}
		cpu := r.CPU
		if cpu == "" {
			cpu = "-"
		}
		mem := r.Memory
		if mem == "" {
			mem = "-"
		}
		drift := r.Drift
		if drift == "" {
			drift = "-"
		}
		fmt.Printf("  %-20s %-10s %-12s %-20s %-18s %-10s %-10s\n",
			r.Name, r.Status, cpu, mem, ip, disk, drift)
	}

	fmt.Println()
	fmt.Printf("  Total: %d resources\n", len(resources))
}

// ResourceStatus статус ресурса для вывода
type ResourceStatus struct {
	Name   string
	Type   string
	Status string
	IP     string
	CPU    string
	Memory string
	Disk   string
	MAC    string
	Drift  string
}

// changeSymbol возвращает символ для типа изменения
func changeSymbol(ct ChangeType) string {
	switch ct {
	case Create:
		return "+"
	case Update:
		return "~"
	case Delete:
		return "-"
	case NoOp:
		return "="
	default:
		return "?"
	}
}

// ConfirmPlan запрашивает подтверждение у пользователя
func ConfirmPlan(plan *Plan) bool {
	fmt.Println("\nПрименить эти изменения? (да/нет)")
	fmt.Print("> ")

	var input string
	fmt.Scanln(&input)

	input = strings.ToLower(strings.TrimSpace(input))
	return input == "да" || input == "y" || input == "yes"
}
