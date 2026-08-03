package libvirt

import (
	"fmt"
	"net/url"
	"time"

	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/socket"
	"github.com/digitalocean/go-libvirt/socket/dialers"
)

// Connection управляет соединением с libvirt
type Connection struct {
	URI       string
	Libvirt   *libvirt.Libvirt
	connected bool
}

// NewConnection создаёт новое соединение
func NewConnection(uri string) *Connection {
	return &Connection{
		URI: uri,
	}
}

// Connect устанавливает соединение с libvirt
func (c *Connection) Connect() error {
	if c.connected {
		return nil
	}

	parsedURI, err := url.Parse(c.URI)
	if err != nil {
		return fmt.Errorf("невалидный URI %s: %w", c.URI, err)
	}

	var dialer socket.Dialer

	switch parsedURI.Scheme {
	case "qemu", "lxc", "test":
		// Локальное соединение через Unix socket
		dialer = dialers.NewLocal()

	case "qemu+ssh", "lxc+ssh":
		// SSH соединение - используем TCP как заглушку
		// Полная SSH поддержка требует настройки ключей
		host := parsedURI.Hostname()
		port := parsedURI.Port()
		if port == "" {
			port = "16509"
		}
		addr := host + ":" + port
		dialer = dialers.NewRemote(addr)

	case "qemu+tcp", "lxc+tcp":
		// TCP соединение
		host := parsedURI.Hostname()
		port := parsedURI.Port()
		if port == "" {
			port = "16509"
		}
		addr := host + ":" + port
		dialer = dialers.NewRemote(addr)

	case "qemu+tls", "lxc+tls":
		// TLS соединение
		host := parsedURI.Hostname()
		port := parsedURI.Port()
		if port == "" {
			port = "16514"
		}
		addr := host + ":" + port
		dialer = dialers.NewTLS(addr)

	default:
		return fmt.Errorf("неподдерживаемая схема URI: %s", parsedURI.Scheme)
	}

	l := libvirt.NewWithDialer(dialer)
	if err := l.Connect(); err != nil {
		return fmt.Errorf("ошибка подключения к libvirt: %w", err)
	}

	c.Libvirt = l
	c.connected = true
	return nil
}

// Disconnect разрывает соединение
func (c *Connection) Disconnect() error {
	if !c.connected || c.Libvirt == nil {
		return nil
	}

	if err := c.Libvirt.Disconnect(); err != nil {
		return fmt.Errorf("ошибка отключения: %w", err)
	}

	c.connected = false
	return nil
}

// IsConnected проверяет активность соединения
func (c *Connection) IsConnected() bool {
	return c.connected
}

// EnsureConnect устанавливает соединение если оно не активно
func (c *Connection) EnsureConnect() error {
	if !c.connected {
		return c.Connect()
	}
	return nil
}

// Ping проверяет соединение
func (c *Connection) Ping() error {
	if err := c.EnsureConnect(); err != nil {
		return err
	}

	_, err := c.Libvirt.ConnectGetVersion()
	return err
}

// GetVersion возвращает версию libvirt
func (c *Connection) GetVersion() (uint64, error) {
	if err := c.EnsureConnect(); err != nil {
		return 0, err
	}
	return c.Libvirt.ConnectGetVersion()
}

// ConnectionPool пул соединений для повторного использования
type ConnectionPool struct {
	connections map[string]*Connection
}

// NewConnectionPool создаёт пул соединений
func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		connections: make(map[string]*Connection),
	}
}

// Get возвращает соединение из пула (или создаёт новое)
func (p *ConnectionPool) Get(uri string) (*Connection, error) {
	if conn, ok := p.connections[uri]; ok {
		if err := conn.EnsureConnect(); err != nil {
			delete(p.connections, uri)
		} else {
			return conn, nil
		}
	}

	conn := NewConnection(uri)
	if err := conn.Connect(); err != nil {
		return nil, err
	}

	p.connections[uri] = conn
	return conn, nil
}

// Close закрывает все соединения в пуле
func (p *ConnectionPool) Close() error {
	var lastErr error
	for uri, conn := range p.connections {
		if err := conn.Disconnect(); err != nil {
			lastErr = fmt.Errorf("ошибка закрытия соединения %s: %w", uri, err)
		}
		delete(p.connections, uri)
	}
	return lastErr
}

// WaitForConnection ожидает доступности libvirt
func WaitForConnection(uri string, timeout time.Duration) (*Connection, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		conn := NewConnection(uri)
		if err := conn.Connect(); err == nil {
			return conn, nil
		}
		time.Sleep(time.Second)
	}

	return nil, fmt.Errorf("тайм-аут ожидания соединения с %s", uri)
}
