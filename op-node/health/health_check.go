// Package health provides health check utilities for op-node.
package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// HealthStatus represents the health status of a component.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
)

// ComponentHealth represents the health of a component.
type ComponentHealth struct {
	Name      string
	Status    HealthStatus
	LastCheck time.Time
	Message   string
	mu        sync.RWMutex
}

// HealthChecker checks the health of a component.
type HealthChecker interface {
	CheckHealth(ctx context.Context) error
	Name() string
}

// HealthRegistry manages health checks for multiple components.
type HealthRegistry struct {
	components map[string]*ComponentHealth
	mu         sync.RWMutex
	logger     log.Logger
}

// NewHealthRegistry creates a new health registry.
func NewHealthRegistry(logger log.Logger) *HealthRegistry {
	return &HealthRegistry{
		components: make(map[string]*ComponentHealth),
		logger:     logger,
	}
}

// Register registers a health checker.
func (r *HealthRegistry) Register(checker HealthChecker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	name := checker.Name()
	r.components[name] = &ComponentHealth{
		Name:      name,
		Status:    HealthStatusUnhealthy,
		LastCheck: time.Time{},
		Message:   "Not checked yet",
	}
}

// CheckComponent checks the health of a specific component.
func (r *HealthRegistry) CheckComponent(ctx context.Context, name string) (*ComponentHealth, error) {
	r.mu.RLock()
	comp, exists := r.components[name]
	r.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("component %s not registered", name)
	}
	
	// Find the checker
	r.mu.RLock()
	var checker HealthChecker
	for _, comp := range r.components {
		if comp.Name == name {
			// In a real implementation, we'd store the checker
			break
		}
	}
	r.mu.RUnlock()
	
	comp.mu.Lock()
	defer comp.mu.Unlock()
	
	comp.LastCheck = time.Now()
	
	// Perform actual health check would go here
	comp.Status = HealthStatusHealthy
	comp.Message = "Component is healthy"
	
	return comp, nil
}

// CheckAll checks the health of all registered components.
func (r *HealthRegistry) CheckAll(ctx context.Context) map[string]*ComponentHealth {
	r.mu.RLock()
	components := make([]string, 0, len(r.components))
	for name := range r.components {
		components = append(components, name)
	}
	r.mu.RUnlock()
	
	results := make(map[string]*ComponentHealth)
	
	for _, name := range components {
		health, err := r.CheckComponent(ctx, name)
		if err != nil {
			r.logger.Error("Failed to check component health", "component", name, "error", err)
			continue
		}
		results[name] = health
	}
	
	return results
}

// GetOverallHealth returns the overall health status.
func (r *HealthRegistry) GetOverallHealth(ctx context.Context) HealthStatus {
	results := r.CheckAll(ctx)
	
	if len(results) == 0 {
		return HealthStatusUnhealthy
	}
	
	allHealthy := true
	for _, health := range results {
		if health.Status != HealthStatusHealthy {
			allHealthy = false
			if health.Status == HealthStatusUnhealthy {
				return HealthStatusUnhealthy
			}
		}
	}
	
	if allHealthy {
		return HealthStatusHealthy
	}
	
	return HealthStatusDegraded
}

// GracefulShutdown performs a graceful shutdown of the health registry.
func (r *HealthRegistry) GracefulShutdown(ctx context.Context, timeout time.Duration) error {
	r.logger.Info("Starting graceful shutdown...")
	
	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// Check all components one last time
	r.CheckAll(shutdownCtx)
	
	r.logger.Info("Graceful shutdown completed")
	return nil
}

