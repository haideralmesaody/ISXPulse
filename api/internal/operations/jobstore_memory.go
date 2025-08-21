package operations

import (
	"fmt"
	"sync"
	"time"
)

// MemoryJobStore is an in-memory implementation of JobStore
type MemoryJobStore struct {
	mu        sync.RWMutex
	jobs      map[string]*Job
	manifests map[string]*PipelineManifest
}

// NewMemoryJobStore creates a new in-memory job store
func NewMemoryJobStore() *MemoryJobStore {
	return &MemoryJobStore{
		jobs:      make(map[string]*Job),
		manifests: make(map[string]*PipelineManifest),
	}
}

// CreateJob creates a new job
func (s *MemoryJobStore) CreateJob(job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.jobs[job.ID]; exists {
		return fmt.Errorf("job %s already exists", job.ID)
	}
	
	s.jobs[job.ID] = job
	return nil
}

// GetJob retrieves a job by ID
func (s *MemoryJobStore) GetJob(id string) (*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	job, exists := s.jobs[id]
	if !exists {
		return nil, fmt.Errorf("job %s not found", id)
	}
	
	// Return a copy to prevent external modification
	jobCopy := *job
	return &jobCopy, nil
}

// UpdateJob updates an existing job
func (s *MemoryJobStore) UpdateJob(job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.jobs[job.ID]; !exists {
		return fmt.Errorf("job %s not found", job.ID)
	}
	
	s.jobs[job.ID] = job
	return nil
}

// ListJobs returns jobs matching the filter
func (s *MemoryJobStore) ListJobs(filter JobFilter) ([]*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var result []*Job
	
	for _, job := range s.jobs {
		// Apply filters
		if filter.Status != "" && job.Status != filter.Status {
			continue
		}
		
		if filter.OperationID != "" && job.OperationID != filter.OperationID {
			continue
		}
		
		if filter.StageID != "" && job.StageID != filter.StageID {
			continue
		}
		
		if !filter.Since.IsZero() && job.CreatedAt.Before(filter.Since) {
			continue
		}
		
		// Make a copy to prevent external modification
		jobCopy := *job
		result = append(result, &jobCopy)
		
		// Apply limit if specified
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}
	
	return result, nil
}

// DeleteJob removes a job from the store
func (s *MemoryJobStore) DeleteJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.jobs[id]; !exists {
		return fmt.Errorf("job %s not found", id)
	}
	
	delete(s.jobs, id)
	return nil
}

// CreateManifest creates a new manifest
func (s *MemoryJobStore) CreateManifest(manifest *PipelineManifest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.manifests[manifest.ID]; exists {
		return fmt.Errorf("manifest %s already exists", manifest.ID)
	}
	
	s.manifests[manifest.ID] = manifest
	return nil
}

// GetManifest retrieves a manifest by ID
func (s *MemoryJobStore) GetManifest(id string) (*PipelineManifest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	manifest, exists := s.manifests[id]
	if !exists {
		return nil, fmt.Errorf("manifest %s not found", id)
	}
	
	// Return a copy to prevent external modification
	manifestCopy := *manifest
	return &manifestCopy, nil
}

// UpdateManifest updates an existing manifest
func (s *MemoryJobStore) UpdateManifest(manifest *PipelineManifest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.manifests[manifest.ID]; !exists {
		return fmt.Errorf("manifest %s not found", manifest.ID)
	}
	
	s.manifests[manifest.ID] = manifest
	return nil
}

// GetManifestByOperationID retrieves a manifest by operation ID
func (s *MemoryJobStore) GetManifestByOperationID(operationID string) (*PipelineManifest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for _, manifest := range s.manifests {
		if manifest.OperationID == operationID {
			// Return a copy to prevent external modification
			manifestCopy := *manifest
			return &manifestCopy, nil
		}
	}
	
	return nil, fmt.Errorf("manifest for operation %s not found", operationID)
}

// CleanupOldJobs removes jobs older than the specified duration
func (s *MemoryJobStore) CleanupOldJobs(olderThan time.Duration) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	cutoff := time.Now().Add(-olderThan)
	deleted := 0
	
	for id, job := range s.jobs {
		// Only delete completed, failed, or cancelled jobs
		if job.Status == JobStatusCompleted || job.Status == JobStatusFailed || job.Status == JobStatusCancelled {
			if job.CreatedAt.Before(cutoff) {
				delete(s.jobs, id)
				deleted++
			}
		}
	}
	
	return deleted, nil
}

// GetStats returns statistics about the job store
func (s *MemoryJobStore) GetStats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	stats := map[string]int{
		"total_jobs":      len(s.jobs),
		"total_manifests": len(s.manifests),
		"pending":         0,
		"running":         0,
		"completed":       0,
		"failed":          0,
		"cancelled":       0,
	}
	
	for _, job := range s.jobs {
		switch job.Status {
		case JobStatusPending:
			stats["pending"]++
		case JobStatusRunning:
			stats["running"]++
		case JobStatusCompleted:
			stats["completed"]++
		case JobStatusFailed:
			stats["failed"]++
		case JobStatusCancelled:
			stats["cancelled"]++
		}
	}
	
	return stats
}