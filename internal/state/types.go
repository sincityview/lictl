package state

import "time"

// ResourceType типы ресурсов
type ResourceType string

const (
	ResourceStorage  ResourceType = "storage"
	ResourceNetwork  ResourceType = "network"
	ResourceDomain   ResourceType = "domain"
)

// ResourceStatus статус ресурса
type ResourceStatus string

const (
	StatusPending   ResourceStatus = "pending"
	StatusCreating  ResourceStatus = "creating"
	StatusRunning   ResourceStatus = "running"
	StatusStopped   ResourceStatus = "stopped"
	StatusDeleting  ResourceStatus = "deleting"
	StatusError     ResourceStatus = "error"
)

// Resource описывает управляемый ресурс
type Resource struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        ResourceType   `json:"type"`
	Status      ResourceStatus `json:"status"`
	ConfigHash  string         `json:"config_hash"`
	LibvirtID   string         `json:"libvirt_id,omitempty"` // UUID в libvirt
	IP          string         `json:"ip,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// StateFile структура файла состояния
type StateFile struct {
	Version   int                `json:"version"`
	Resources []Resource         `json:"resources"`
	Metadata  StateMetadata      `json:"metadata"`
}

// StateMetadata метаданные состояния
type StateMetadata struct {
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ProviderURI string    `json:"provider_uri"`
}

// NewResource создаёт новый ресурс
func NewResource(id, name string, rtype ResourceType) *Resource {
	now := time.Now()
	return &Resource{
		ID:        id,
		Name:      name,
		Type:      rtype,
		Status:    StatusPending,
		Metadata:  make(map[string]string),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// UpdateStatus обновляет статус ресурса
func (r *Resource) UpdateStatus(status ResourceStatus) {
	r.Status = status
	r.UpdatedAt = time.Now()
}

// SetLibvirtID устанавливает ID в libvirt
func (r *Resource) SetLibvirtID(id string) {
	r.LibvirtID = id
	r.UpdatedAt = time.Now()
}

// SetIP устанавливает IP-адрес
func (r *Resource) SetIP(ip string) {
	r.IP = ip
	r.UpdatedAt = time.Now()
}

// SetConfigHash устанавливает хеш конфигурации
func (r *Resource) SetConfigHash(hash string) {
	r.ConfigHash = hash
	r.UpdatedAt = time.Now()
}
