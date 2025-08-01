package services

import (
	"context"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

func TestNewPullJobService(t *testing.T) {
	cfg := &config.Config{
		JobTimeout: 30 * time.Second,
		MaxRetries: 3,
	}

	dpService := NewDPConnectorService(cfg)
	service := NewPullJobService(cfg, dpService)

	if service == nil {
		t.Fatal("Expected service to be created")
	}

	if service.config != cfg {
		t.Error("Expected config to be set")
	}

	if service.dpService != dpService {
		t.Error("Expected DP service to be set")
	}

	if service.jobTracker == nil {
		t.Error("Expected job tracker to be created")
	}

	if service.auditLogger == nil {
		t.Error("Expected audit logger to be created")
	}
}

func TestPullJobService_SubmitJob(t *testing.T) {
	cfg := &config.Config{
		JobTimeout: 30 * time.Second,
		MaxRetries: 3,
	}

	dpService := NewDPConnectorService(cfg)
	service := NewPullJobService(cfg, dpService)

	req := &models.PrivacyRequest{
		RPID:      "rp_123",
		UserHash:  "hash_abc123",
		ClaimType: "student_verification",
		HashedIdentifiers: map[string]string{
			"student_id": "hash_student_123",
		},
		BloomFilters: map[string]string{
			"student_id": "bloom_filter_data",
		},
	}

	ctx := context.Background()
	jobStatus, err := service.SubmitJob(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if jobStatus == nil {
		t.Fatal("Expected job status to be returned")
	}

	if jobStatus.JobID == "" {
		t.Error("Expected job ID to be generated")
	}

	if jobStatus.RequestID == "" {
		t.Error("Expected request ID to be set")
	}

	if jobStatus.Status != JobPending {
		t.Errorf("Expected status to be pending, got %s", jobStatus.Status)
	}

	if jobStatus.CreatedAt.IsZero() {
		t.Error("Expected created at to be set")
	}

	if jobStatus.UpdatedAt.IsZero() {
		t.Error("Expected updated at to be set")
	}
}

func TestPullJobService_GetJobStatus(t *testing.T) {
	cfg := &config.Config{
		JobTimeout: 30 * time.Second,
		MaxRetries: 3,
	}

	dpService := NewDPConnectorService(cfg)
	service := NewPullJobService(cfg, dpService)

	req := &models.PrivacyRequest{
		RPID:      "rp_123",
		UserHash:  "hash_abc123",
		ClaimType: "student_verification",
	}

	ctx := context.Background()
	jobStatus, err := service.SubmitJob(ctx, req)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Get job status
	retrievedStatus, err := service.GetJobStatus(jobStatus.JobID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if retrievedStatus == nil {
		t.Fatal("Expected job status to be returned")
	}

	if retrievedStatus.JobID != jobStatus.JobID {
		t.Errorf("Expected job ID %s, got %s", jobStatus.JobID, retrievedStatus.JobID)
	}
}

func TestPullJobService_GetJobStatus_NotFound(t *testing.T) {
	cfg := &config.Config{
		JobTimeout: 30 * time.Second,
		MaxRetries: 3,
	}

	dpService := NewDPConnectorService(cfg)
	service := NewPullJobService(cfg, dpService)

	_, err := service.GetJobStatus("nonexistent_job_id")
	if err == nil {
		t.Error("Expected error for non-existent job")
	}
}

func TestPullJobService_ListJobs(t *testing.T) {
	cfg := &config.Config{
		JobTimeout: 30 * time.Second,
		MaxRetries: 3,
	}

	dpService := NewDPConnectorService(cfg)
	service := NewPullJobService(cfg, dpService)

	// Submit multiple jobs
	req1 := &models.PrivacyRequest{
		RPID:      "rp_123",
		UserHash:  "hash_abc123",
		ClaimType: "student_verification",
	}

	req2 := &models.PrivacyRequest{
		RPID:      "rp_456",
		UserHash:  "hash_def456",
		ClaimType: "age_verification",
	}

	ctx := context.Background()
	service.SubmitJob(ctx, req1)
	service.SubmitJob(ctx, req2)

	jobs := service.ListJobs()
	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(jobs))
	}
}

func TestJobTracker_TrackJob(t *testing.T) {
	tracker := &JobTracker{
		jobs: make(map[string]*JobStatus),
	}

	job := &JobStatus{
		JobID:     "job_123",
		RequestID: "req_123",
		Status:    JobPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tracker.TrackJob(job)

	if len(tracker.jobs) != 1 {
		t.Errorf("Expected 1 job, got %d", len(tracker.jobs))
	}

	if tracker.jobs["job_123"] != job {
		t.Error("Expected job to be tracked")
	}
}

func TestJobTracker_UpdateJobStatus(t *testing.T) {
	tracker := &JobTracker{
		jobs: make(map[string]*JobStatus),
	}

	job := &JobStatus{
		JobID:     "job_123",
		RequestID: "req_123",
		Status:    JobPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tracker.TrackJob(job)

	// Update job status
	tracker.UpdateJobStatus("job_123", JobRunning, "")

	updatedJob := tracker.jobs["job_123"]
	if updatedJob.Status != JobRunning {
		t.Errorf("Expected status to be running, got %s", updatedJob.Status)
	}

	if updatedJob.UpdatedAt.Equal(job.UpdatedAt) {
		t.Error("Expected updated at to be changed")
	}
}

func TestJobTracker_CompleteJob(t *testing.T) {
	tracker := &JobTracker{
		jobs: make(map[string]*JobStatus),
	}

	job := &JobStatus{
		JobID:     "job_123",
		RequestID: "req_123",
		Status:    JobRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tracker.TrackJob(job)

	result := &models.DPResponse{
		JobID:  "job_123",
		Status: "completed",
		VerificationResult: &VerificationResult{
			Verified:   true,
			Confidence: 0.95,
			Reason:     "Student ID found",
			Timestamp:  "2025-08-02T07:00:00Z",
		},
	}

	tracker.CompleteJob("job_123", result, "")

	completedJob := tracker.jobs["job_123"]
	if completedJob.Status != JobCompleted {
		t.Errorf("Expected status to be completed, got %s", completedJob.Status)
	}

	if completedJob.Result == nil {
		t.Error("Expected result to be set")
	}

	if completedJob.CompletedAt == nil {
		t.Error("Expected completed at to be set")
	}
}

func TestJobTracker_GetJob(t *testing.T) {
	tracker := &JobTracker{
		jobs: make(map[string]*JobStatus),
	}

	job := &JobStatus{
		JobID:     "job_123",
		RequestID: "req_123",
		Status:    JobPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tracker.TrackJob(job)

	retrievedJob, err := tracker.GetJob("job_123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if retrievedJob != job {
		t.Error("Expected job to be retrieved")
	}
}

func TestJobTracker_GetJob_NotFound(t *testing.T) {
	tracker := &JobTracker{
		jobs: make(map[string]*JobStatus),
	}

	_, err := tracker.GetJob("nonexistent_job")
	if err == nil {
		t.Error("Expected error for non-existent job")
	}
}

func TestJobAuditLogger_LogEvent(t *testing.T) {
	logger := &JobAuditLogger{
		events: make([]JobAuditEvent, 0),
	}

	metadata := map[string]string{
		"user_id": "user_123",
		"rp_id":   "rp_123",
	}

	logger.LogEvent("job_created", "Job submitted successfully", "job_123", "req_123", JobPending, metadata)

	if len(logger.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(logger.events))
	}

	event := logger.events[0]
	if event.EventType != "job_created" {
		t.Errorf("Expected event type 'job_created', got %s", event.EventType)
	}

	if event.JobID != "job_123" {
		t.Errorf("Expected job ID 'job_123', got %s", event.JobID)
	}

	if event.RequestID != "req_123" {
		t.Errorf("Expected request ID 'req_123', got %s", event.RequestID)
	}

	if event.Status != JobPending {
		t.Errorf("Expected status pending, got %s", event.Status)
	}
}

func TestJobAuditLogger_GetAuditEvents(t *testing.T) {
	logger := &JobAuditLogger{
		events: make([]JobAuditEvent, 0),
	}

	// Log multiple events
	logger.LogEvent("job_created", "Job submitted", "job_123", "req_123", JobPending, nil)
	logger.LogEvent("job_started", "Job started processing", "job_123", "req_123", JobRunning, nil)
	logger.LogEvent("job_completed", "Job completed successfully", "job_123", "req_123", JobCompleted, nil)

	events := logger.GetAuditEvents()
	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}
}

func TestPullJobService_GenerateJobID(t *testing.T) {
	cfg := &config.Config{
		JobTimeout: 30 * time.Second,
		MaxRetries: 3,
	}

	dpService := NewDPConnectorService(cfg)
	service := NewPullJobService(cfg, dpService)

	jobID1 := service.generateJobID()
	jobID2 := service.generateJobID()

	if jobID1 == "" {
		t.Error("Expected job ID to be generated")
	}

	if jobID2 == "" {
		t.Error("Expected job ID to be generated")
	}

	if jobID1 == jobID2 {
		t.Error("Expected job IDs to be unique")
	}
}

func TestPullJobService_CleanupExpiredJobs(t *testing.T) {
	cfg := &config.Config{
		JobTimeout: 30 * time.Second,
		MaxRetries: 3,
	}

	dpService := NewDPConnectorService(cfg)
	service := NewPullJobService(cfg, dpService)

	// Create an expired job
	expiredJob := &JobStatus{
		JobID:     "expired_job",
		RequestID: "req_123",
		Status:    JobCompleted,
		CreatedAt: time.Now().Add(-2 * time.Hour), // 2 hours ago
		UpdatedAt: time.Now().Add(-2 * time.Hour),
		CompletedAt: func() *time.Time {
			t := time.Now().Add(-2 * time.Hour)
			return &t
		}(),
	}

	service.jobTracker.TrackJob(expiredJob)

	// Create a recent job
	recentJob := &JobStatus{
		JobID:     "recent_job",
		RequestID: "req_456",
		Status:    JobPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	service.jobTracker.TrackJob(recentJob)

	// Cleanup expired jobs
	service.CleanupExpiredJobs()

	jobs := service.ListJobs()
	if len(jobs) != 1 {
		t.Errorf("Expected 1 job after cleanup, got %d", len(jobs))
	}

	if jobs[0].JobID != "recent_job" {
		t.Error("Expected recent job to remain after cleanup")
	}
}

func TestPullJobService_GetJobStats(t *testing.T) {
	cfg := &config.Config{
		JobTimeout: 30 * time.Second,
		MaxRetries: 3,
	}

	dpService := NewDPConnectorService(cfg)
	service := NewPullJobService(cfg, dpService)

	// Submit some jobs
	req := &models.PrivacyRequest{
		RPID:      "rp_123",
		UserHash:  "hash_abc123",
		ClaimType: "student_verification",
	}

	ctx := context.Background()
	service.SubmitJob(ctx, req)
	service.SubmitJob(ctx, req)

	stats := service.GetJobStats()

	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}

	// Check that stats contain expected fields
	expectedFields := []string{"total_jobs", "pending_jobs", "completed_jobs", "failed_jobs", "audit_events"}
	for _, field := range expectedFields {
		if _, exists := stats[field]; !exists {
			t.Errorf("Expected stats to contain field: %s", field)
		}
	}
}

func TestPullJobService_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		JobTimeout: 30 * time.Second,
		MaxRetries: 3,
	}

	dpService := NewDPConnectorService(cfg)
	service := NewPullJobService(cfg, dpService)

	ctx := context.Background()
	err := service.HealthCheck(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
} 