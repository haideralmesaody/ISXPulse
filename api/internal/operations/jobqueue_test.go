package operations

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobQueue(t *testing.T) {
	t.Run("basic job execution", func(t *testing.T) {
		// Create test infrastructure
		store := NewMemoryJobStore()
		registry := NewRegistry()
		config := NewConfig()
		manager := NewManager(nil, registry, config)
		
		// Create job queue
		queue := NewJobQueue(2, store, manager, nil)
		
		// Start queue
		ctx := context.Background()
		queue.Start(ctx)
		defer queue.Stop(5 * time.Second)
		
		// Create a test job
		job := &Job{
			ID:          "test-job-1",
			OperationID: "op-1",
			StageID:     "test-stage",
			StageName:   "Test Stage",
			Status:      JobStatusPending,
			CreatedAt:   time.Now(),
			Metadata:    map[string]interface{}{"test": true},
		}
		
		// Enqueue job
		err := queue.Enqueue(job)
		require.NoError(t, err)
		
		// Wait a bit for processing
		time.Sleep(100 * time.Millisecond)
		
		// Check job status
		retrievedJob, err := queue.GetJob(job.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrievedJob)
		
		// Job should either be running or failed (since we don't have a real stage)
		assert.Contains(t, []JobStatus{JobStatusRunning, JobStatusFailed}, retrievedJob.Status)
	})
	
	t.Run("multiple jobs", func(t *testing.T) {
		// Create test infrastructure
		store := NewMemoryJobStore()
		registry := NewRegistry()
		config := NewConfig()
		manager := NewManager(nil, registry, config)
		
		// Create job queue with limited workers
		queue := NewJobQueue(1, store, manager, nil)
		
		// Start queue
		ctx := context.Background()
		queue.Start(ctx)
		defer queue.Stop(5 * time.Second)
		
		// Create multiple jobs
		jobs := []*Job{
			{
				ID:          "test-job-2",
				OperationID: "op-2",
				StageID:     "stage-1",
				StageName:   "Stage 1",
				CreatedAt:   time.Now(),
			},
			{
				ID:          "test-job-3",
				OperationID: "op-3",
				StageID:     "stage-2",
				StageName:   "Stage 2",
				CreatedAt:   time.Now(),
			},
			{
				ID:          "test-job-4",
				OperationID: "op-4",
				StageID:     "stage-3",
				StageName:   "Stage 3",
				CreatedAt:   time.Now(),
			},
		}
		
		// Enqueue all jobs
		for _, job := range jobs {
			err := queue.Enqueue(job)
			require.NoError(t, err)
		}
		
		// Wait for processing
		time.Sleep(200 * time.Millisecond)
		
		// List all jobs
		allJobs, err := queue.ListJobs(JobFilter{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(allJobs), 3)
	})
	
	t.Run("job cancellation", func(t *testing.T) {
		// Create test infrastructure
		store := NewMemoryJobStore()
		registry := NewRegistry()
		config := NewConfig()
		manager := NewManager(nil, registry, config)
		
		// Create job queue
		queue := NewJobQueue(1, store, manager, nil)
		
		// Don't start the queue yet
		
		// Create a job
		job := &Job{
			ID:          "test-job-5",
			OperationID: "op-5",
			StageID:     "cancel-stage",
			StageName:   "Cancel Stage",
			Status:      JobStatusPending,
			CreatedAt:   time.Now(),
		}
		
		// Add job directly to store
		err := store.CreateJob(job)
		require.NoError(t, err)
		
		// Cancel the job
		err = queue.CancelJob(job.ID)
		require.NoError(t, err)
		
		// Check job status
		cancelledJob, err := store.GetJob(job.ID)
		require.NoError(t, err)
		assert.Equal(t, JobStatusCancelled, cancelledJob.Status)
	})
	
	t.Run("queue statistics", func(t *testing.T) {
		// Create test infrastructure
		store := NewMemoryJobStore()
		registry := NewRegistry()
		config := NewConfig()
		manager := NewManager(nil, registry, config)
		
		// Create job queue
		queue := NewJobQueue(2, store, manager, nil)
		
		// Get stats
		stats := queue.GetQueueStats()
		assert.NotNil(t, stats)
		assert.Equal(t, 2, stats["workers"])
		assert.GreaterOrEqual(t, stats["queue_cap"].(int), 4) // Should be at least 2*workers
	})
}

func TestMemoryJobStore(t *testing.T) {
	t.Run("job CRUD operations", func(t *testing.T) {
		store := NewMemoryJobStore()
		
		// Create job
		job := &Job{
			ID:          "store-test-1",
			OperationID: "op-store-1",
			StageID:     "stage-store",
			Status:      JobStatusPending,
			CreatedAt:   time.Now(),
		}
		
		// Create
		err := store.CreateJob(job)
		require.NoError(t, err)
		
		// Read
		retrieved, err := store.GetJob(job.ID)
		require.NoError(t, err)
		assert.Equal(t, job.ID, retrieved.ID)
		assert.Equal(t, job.OperationID, retrieved.OperationID)
		
		// Update
		job.Status = JobStatusRunning
		err = store.UpdateJob(job)
		require.NoError(t, err)
		
		retrieved, err = store.GetJob(job.ID)
		require.NoError(t, err)
		assert.Equal(t, JobStatusRunning, retrieved.Status)
		
		// Delete
		err = store.DeleteJob(job.ID)
		require.NoError(t, err)
		
		_, err = store.GetJob(job.ID)
		assert.Error(t, err)
	})
	
	t.Run("job filtering", func(t *testing.T) {
		store := NewMemoryJobStore()
		
		// Create multiple jobs
		jobs := []*Job{
			{
				ID:          "filter-1",
				OperationID: "op-filter",
				StageID:     "stage-a",
				Status:      JobStatusPending,
				CreatedAt:   time.Now(),
			},
			{
				ID:          "filter-2",
				OperationID: "op-filter",
				StageID:     "stage-b",
				Status:      JobStatusRunning,
				CreatedAt:   time.Now(),
			},
			{
				ID:          "filter-3",
				OperationID: "op-other",
				StageID:     "stage-a",
				Status:      JobStatusCompleted,
				CreatedAt:   time.Now(),
			},
		}
		
		for _, job := range jobs {
			err := store.CreateJob(job)
			require.NoError(t, err)
		}
		
		// Filter by operation ID
		filtered, err := store.ListJobs(JobFilter{OperationID: "op-filter"})
		require.NoError(t, err)
		assert.Len(t, filtered, 2)
		
		// Filter by status
		filtered, err = store.ListJobs(JobFilter{Status: JobStatusRunning})
		require.NoError(t, err)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "filter-2", filtered[0].ID)
		
		// Filter by stage ID
		filtered, err = store.ListJobs(JobFilter{StageID: "stage-a"})
		require.NoError(t, err)
		assert.Len(t, filtered, 2)
		
		// Filter with limit
		filtered, err = store.ListJobs(JobFilter{Limit: 2})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(filtered), 2)
	})
	
	t.Run("manifest operations", func(t *testing.T) {
		store := NewMemoryJobStore()
		
		// Create manifest
		manifest := NewPipelineManifest("op-manifest", "2025-01-01", "2025-01-31")
		
		// Create
		err := store.CreateManifest(manifest)
		require.NoError(t, err)
		
		// Read by ID
		retrieved, err := store.GetManifest(manifest.ID)
		require.NoError(t, err)
		assert.Equal(t, manifest.ID, retrieved.ID)
		assert.Equal(t, manifest.OperationID, retrieved.OperationID)
		
		// Read by operation ID
		retrieved, err = store.GetManifestByOperationID("op-manifest")
		require.NoError(t, err)
		assert.Equal(t, manifest.ID, retrieved.ID)
		
		// Update
		manifest.Status = "running"
		err = store.UpdateManifest(manifest)
		require.NoError(t, err)
		
		retrieved, err = store.GetManifest(manifest.ID)
		require.NoError(t, err)
		assert.Equal(t, "running", retrieved.Status)
	})
	
	t.Run("cleanup old jobs", func(t *testing.T) {
		store := NewMemoryJobStore()
		
		// Create old completed job
		oldJob := &Job{
			ID:          "old-job",
			Status:      JobStatusCompleted,
			CreatedAt:   time.Now().Add(-2 * time.Hour),
		}
		err := store.CreateJob(oldJob)
		require.NoError(t, err)
		
		// Create recent job
		recentJob := &Job{
			ID:        "recent-job",
			Status:    JobStatusCompleted,
			CreatedAt: time.Now(),
		}
		err = store.CreateJob(recentJob)
		require.NoError(t, err)
		
		// Create running job (should not be deleted)
		runningJob := &Job{
			ID:        "running-job",
			Status:    JobStatusRunning,
			CreatedAt: time.Now().Add(-3 * time.Hour),
		}
		err = store.CreateJob(runningJob)
		require.NoError(t, err)
		
		// Cleanup jobs older than 1 hour
		deleted, err := store.CleanupOldJobs(1 * time.Hour)
		require.NoError(t, err)
		assert.Equal(t, 1, deleted) // Only old completed job should be deleted
		
		// Verify correct jobs remain
		_, err = store.GetJob("old-job")
		assert.Error(t, err) // Should be deleted
		
		_, err = store.GetJob("recent-job")
		assert.NoError(t, err) // Should exist
		
		_, err = store.GetJob("running-job")
		assert.NoError(t, err) // Should exist (not deleted because running)
	})
}