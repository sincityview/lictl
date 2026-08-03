package config

import (
	"reflect"
	"testing"
)

func TestExpandRanges(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single value",
			input:    "test-vm",
			expected: []string{"test-vm"},
		},
		{
			name:     "range 1-3",
			input:    "worker-{1..3}",
			expected: []string{"worker-1", "worker-2", "worker-3"},
		},
		{
			name:     "range 0-2",
			input:    "node-{0..2}",
			expected: []string{"node-0", "node-1", "node-2"},
		},
		{
			name:     "reversed range",
			input:    "test-{3..1}",
			expected: []string{"test-1", "test-2", "test-3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandRanges(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ExpandRanges() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExpandVMConfig(t *testing.T) {
	vm := VMConfig{
		Name:   "worker-{1..3}",
		CPU:    2,
		Memory: 2048,
	}

	result := ExpandVMConfig(vm)

	if len(result) != 3 {
		t.Fatalf("ожидалось 3 VM, получено %d", len(result))
	}

	expectedNames := []string{"worker-1", "worker-2", "worker-3"}
	for i, name := range expectedNames {
		if result[i].Name != name {
			t.Errorf("VM %d: ожидалось имя '%s', получено '%s'", i, name, result[i].Name)
		}
		if result[i].CPU != 2 {
			t.Errorf("VM %d: ожидался CPU 2, получено %d", i, result[i].CPU)
		}
	}
}

func TestExpandVMConfigWithHostname(t *testing.T) {
	vm := VMConfig{
		Name:   "worker-{1..2}",
		CPU:    2,
		Memory: 2048,
		CloudInit: &CloudInit{
			Hostname: "worker-{N}",
		},
	}

	result := ExpandVMConfig(vm)

	if len(result) != 2 {
		t.Fatalf("ожидалось 2 VM, получено %d", len(result))
	}

	if result[0].CloudInit.Hostname != "worker-1" {
		t.Errorf("ожидался hostname 'worker-1', получено '%s'", result[0].CloudInit.Hostname)
	}
	if result[1].CloudInit.Hostname != "worker-2" {
		t.Errorf("ожидался hostname 'worker-2', получено '%s'", result[1].CloudInit.Hostname)
	}
}

func TestExpandAllVMs(t *testing.T) {
	vms := []VMConfig{
		{Name: "control-plane", CPU: 2, Memory: 4096},
		{Name: "worker-{1..3}", CPU: 4, Memory: 8192},
	}

	result := ExpandAllVMs(vms)

	if len(result) != 4 {
		t.Fatalf("ожидалось 4 VM, получено %d", len(result))
	}

	expectedNames := []string{"control-plane", "worker-1", "worker-2", "worker-3"}
	for i, name := range expectedNames {
		if result[i].Name != name {
			t.Errorf("VM %d: ожидалось имя '%s', получено '%s'", i, name, result[i].Name)
		}
	}
}
