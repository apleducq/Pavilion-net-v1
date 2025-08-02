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
		DPTimeout: 30 * time.Second,
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
		DPTimeout: 30 * time.Second,
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
		DPTimeout: 30 * time.Second,
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
		t.Error("Expected job ID to match")
	}

	if retrievedStatus.Status != JobPending {
		t.Errorf("Expected status to be pending, got %s", retrievedStatus.Status)
	}
}

func TestPullJobService_GetJobStatus_NotFound(t *testing.T) {
	cfg := &config.Config{
		DPTimeout: 30 * time.Second,
	}

	dpService := NewDPConnectorService(cfg)
	service := NewPullJobService(cfg, dpService)

	// Try to get non-existent job
	_, err := service.GetJobStatus("non-existent-job")
	if err == nil {
		t.Error("Expected error for non-existent job")
	}
}

func TestPullJobService_ListJobs(t *testing.T) {
	cfg := &config.Config{
		DPTimeout: 30 * time.Second,
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

	// Complete job with result
	result := &models.DPResponse{
		Status:          "verified",
		Verified:        true,
		ConfidenceScore: 0.95,
		Reason:          "Student enrollment confirmed",
		DPID:            "dp_university_001",
		Timestamp:       time.Now().Format(time.RFC3339),
	}

	tracker.CompleteJob("job_123", result, "")

	completedJob := tracker.jobs["job_123"]
	if completedJob.Status != JobCompleted {
		t.Errorf("Expected status to be completed, got %s", completedJob.Status)
	}

	if completedJob.Result == nil {
		t.Error("Expected result to be set")
	}

	if completedJob.Result.Status != "verified" {
		t.Errorf("Expected result status to be verified, got %s", completedJob.Result.Status)
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

	// Get job
	retrievedJob, err := tracker.GetJob("job_123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if retrievedJob == nil {
		t.Fatal("Expected job to be returned")
	}

	if retrievedJob.JobID != "job_123" {
		t.Error("Expected job ID to match")
	}
}

func TestJobTracker_GetJob_NotFound(t *testing.T) {
	tracker := &JobTracker{
		jobs: make(map[string]*JobStatus),
	}

	// Try to get non-existent job
	job, err := tracker.GetJob("non-existent-job")
	if err == nil {
		t.Error("Expected error for non-existent job")
	}
	if job != nil {
		t.Error("Expected nil for non-existent job")
	}
}

func TestJobAuditLogger_LogEvent(t *testing.T) {
	logger := &JobAuditLogger{
		events: make([]JobAuditEvent, 0),
	}

	eventType := "job_submitted"
	description := "Job submitted successfully"
	jobID := "job_123"
	requestID := "req_123"
	metadata := map[string]string{
		"rp_id":      "rp_123",
		"claim_type": "student_verification",
	}

	logger.LogEvent(eventType, description, jobID, requestID, JobPending, metadata)

	events := logger.GetAuditEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.EventType != eventType {
		t.Errorf("Expected event type %s, got %s", eventType, event.EventType)
	}

	if event.JobID != jobID {
		t.Errorf("Expected job ID %s, got %s", jobID, event.JobID)
	}

	if event.RequestID != requestID {
		t.Errorf("Expected request ID %s, got %s", requestID, event.RequestID)
	}
}

func TestJobAuditLogger_GetAuditEvents(t *testing.T) {
	logger := &JobAuditLogger{
		events: make([]JobAuditEvent, 0),
	}

	// Add multiple events
	logger.LogEvent("job_submitted", "Event 1", "job1", "req1", JobPending, nil)
	logger.LogEvent("job_completed", "Event 2", "job2", "req2", JobCompleted, nil)
	logger.LogEvent("job_failed", "Event 3", "job3", "req3", JobFailed, nil)

	events := logger.GetAuditEvents()
	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}
}

func TestPullJobService_GenerateJobID(t *testing.T) {
	cfg := &config.Config{
		DPTimeout: 30 * time.Second,
	}

	dpService := NewDPConnectorService(cfg)
	service := NewPullJobService(cfg, dpService)

	// Generate job IDs
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

	// Check format
	if len(jobID1) < 10 {
		t.Error("Expected job ID to have reasonable length")
	}
}

func TestPullJobService_CleanupExpiredJobs(t *testing.T) {
	cfg := &config.Config{
		DPTimeout: 30 * time.Second,
	}

	dpService := NewDPConnectorService(cfg)
	service := NewPullJobService(cfg, dpService)

	// Submit a job
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

	// Manually set job to expired
	jobStatus.CreatedAt = time.Now().Add(-25 * time.Hour) // Expired
	service.jobTracker.jobs[jobStatus.JobID] = jobStatus

	// Cleanup expired jobs
	service.CleanupExpiredJobs()

	// Check that job was removed
	_, err = service.GetJobStatus(jobStatus.JobID)
	if err == nil {
		t.Error("Expected job to be removed")
	}
}

func TestPullJobService_GetJobStats(t *testing.T) {
	cfg := &config.Config{
		DPTimeout: 30 * time.Second,
	}

	dpService := NewDPConnectorService(cfg)
	service := NewPullJobService(cfg, dpService)

	// Submit some jobs
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

	// Get stats
	stats := service.GetJobStats()

	if stats["total_jobs"] != 2 {
		t.Errorf("Expected 2 total jobs, got %v", stats["total_jobs"])
	}

	if stats["pending_jobs"] != 2 {
		t.Errorf("Expected 2 pending jobs, got %v", stats["pending_jobs"])
	}
}

func TestPullJobService_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		DPTimeout: 30 * time.Second,
	}

	dpService := NewDPConnectorService(cfg)
	service := NewPullJobService(cfg, dpService)

	ctx := context.Background()
	err := service.HealthCheck(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
