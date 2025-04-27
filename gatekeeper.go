package gatekeeper

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Common resource types
const (
	ResourceUser   = "user"
	ResourceReward = "reward"
)

// Config represents the gatekeeper configuration
type Config struct {
	LockTimeout         time.Duration
	Enabled             bool
	DefaultErrorStatus  int
	DefaultErrorMessage string
	DefaultErrorCode    string
}

// Option function type for functional options pattern
type Option func(*Config)

// gatekeeper maintains the internal state
type gatekeeper struct {
	locks  map[string]map[string]time.Time
	mutex  sync.RWMutex
	config Config
}

// singleton instance
var instance = &gatekeeper{
	locks: make(map[string]map[string]time.Time),
	config: Config{
		LockTimeout:         5 * time.Second,
		Enabled:             true,
		DefaultErrorStatus:  fiber.StatusTooManyRequests,
		DefaultErrorMessage: "Resource is currently being processed",
		DefaultErrorCode:    "RESOURCE_LOCKED",
	},
}

// WithLockTimeout Option functions
func WithLockTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.LockTimeout = timeout
	}
}

func WithEnabled(enabled bool) Option {
	return func(c *Config) {
		c.Enabled = enabled
	}
}

func WithErrorStatus(status int) Option {
	return func(c *Config) {
		c.DefaultErrorStatus = status
	}
}

func WithErrorMessage(message string) Option {
	return func(c *Config) {
		c.DefaultErrorMessage = message
	}
}

func WithErrorCode(code string) Option {
	return func(c *Config) {
		c.DefaultErrorCode = code
	}
}

// Setup configures the gatekeeper with optional configuration overrides
func Setup(opts ...Option) {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	for _, opt := range opts {
		opt(&instance.config)
	}
}

// TryLock attempts to acquire a lock for a resource
func TryLock(resourceType, resourceID string) bool {
	if !instance.config.Enabled {
		return true
	}

	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	// Initialize a resource type map if needed
	if instance.locks[resourceType] == nil {
		instance.locks[resourceType] = make(map[string]time.Time)
	}

	// Check if the resource is locked and if the lock has expired
	if timestamp, exists := instance.locks[resourceType][resourceID]; exists {
		if time.Since(timestamp) < instance.config.LockTimeout {
			return false
		}
	}

	// Acquire the lock
	instance.locks[resourceType][resourceID] = time.Now()
	return true
}

// ReleaseLock releases a lock on a resource
func ReleaseLock(resourceType, resourceID string) {
	if !instance.config.Enabled {
		return
	}

	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	if locks, exists := instance.locks[resourceType]; exists {
		delete(locks, resourceID)
	}
}

// IsLocked checks if a resource is currently locked
func IsLocked(resourceType, resourceID string) bool {
	if !instance.config.Enabled {
		return false
	}

	instance.mutex.RLock()
	defer instance.mutex.RUnlock()

	locks, exists := instance.locks[resourceType]
	if !exists {
		return false
	}

	timestamp, exists := locks[resourceID]
	if !exists {
		return false
	}

	return time.Since(timestamp) < instance.config.LockTimeout
}

// Middleware creates a Fiber middleware that protects routes using resource locking
func Middleware(resourceType string, idExtractor func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !instance.config.Enabled {
			return c.Next()
		}

		resourceID := idExtractor(c)
		if resourceID == "" {
			return c.Next()
		}

		if !TryLock(resourceType, resourceID) {
			return c.Status(instance.config.DefaultErrorStatus).JSON(fiber.Map{
				"message": instance.config.DefaultErrorMessage,
				"code":    instance.config.DefaultErrorCode,
			})
		}

		defer ReleaseLock(resourceType, resourceID)
		return c.Next()
	}
}
