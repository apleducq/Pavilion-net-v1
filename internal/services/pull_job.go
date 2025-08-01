package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

// PullJobService handles pull-job protocol communication
type PullJobService struct {
	config *config.Config
	dpService *DPConnectorService
	// Job tracking
	jobTracker *JobTracker
	// Audit logger for job events
	auditLogger *JobAuditLogger
}

// JobTracker tracks job status and results
type JobTracker struct {
	mu    sync.RWMutex
	jobs  map[string]*JobStatus
}

// JobStatus represents the status of a pull-job
type JobStatus struct {
	JobID           string                 `json:"job_id"`
	RequestID       string                 `json:"request_id"`
	Status          JobState               `json:"status"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	Result          *models.DPResponse     `json:"result,omitempty"`
	Error           string                 `json:"error,omitempty"`
	RetryCount      int                    `json:"retry_count"`
	MaxRetries      int                    `json:"max_retries"`
	Timeout         time.Duration          `json:"timeout"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// JobState represents the state of a job
type JobState string

const (
	JobPending   JobState = "pending"
	JobRunning   JobState = "running"
	JobCompleted JobState = "completed"
	JobFailed    JobState = "failed"
	JobTimeout   JobState = "timeout"
)

// JobAuditLogger handles job-related audit logging
type JobAuditLogger struct {
	mu     sync.Mutex
	events []JobAuditEvent
}

// JobAuditEvent represents a job audit event
type JobAuditEvent struct {
	Timestamp   string            `json:"timestamp"`
	EventType   string            `json:"event_type"`
	JobID       string            `json:"job_id"`
	RequestID   string            `json:"request_id"`
	Description string            `json:"description"`
	Status      JobState          `json:"status"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// PullJobRequest represents a pull-job request
type PullJobRequest struct {
	RPID             string                 `json:"rp_id"`
	UserHash         string                 `json:"user_hash"`
	ClaimType        string                 `json:"claim_type"`
	HashedIdentifiers map[string]string    `json:"hashed_identifiers"`
	BloomFilters     map[string]string     `json:"bloom_filters"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// NewPullJobService creates a new pull-job service
func NewPullJobService(cfg *config.Config, dpService *DPConnectorService) *PullJobService {
	return &PullJobService{
		config:    cfg,
		dpService: dpService,
		jobTracker: &JobTracker{
			jobs: make(map[string]*JobStatus),
		},
		auditLogger: &JobAuditLogger{
			events: make([]JobAuditEvent, 0),
		},
	}
}

// SubmitJob submits a new pull-job request
func (s *PullJobService) SubmitJob(ctx context.Context, req *models.PrivacyRequest) (*JobStatus, error) {
	// Generate job ID
	jobID := s.generateJobID()
	requestID := req.Metadata["request_id"].(string)

	// Create job status
	jobStatus := &JobStatus{
		JobID:      jobID,
		RequestID:  requestID,
		Status:     JobPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		RetryCount: 0,
		MaxRetries: 3,
		Timeout:    5 * time.Minute,
		Metadata:   req.Metadata,
	}

	// Track the job
	s.jobTracker.TrackJob(jobStatus)

	// Log job submission
	s.auditLogger.LogEvent("job_submitted", "Pull-job submitted", jobID, requestID, JobPending, map[string]string{
		"rp_id":      req.RPID,
		"claim_type": req.ClaimType,
	})

	// Start job processing in background
	go s.processJob(ctx, jobStatus, req)

	return jobStatus, nil
}

// processJob processes a job in the background
func (s *PullJobService) processJob(ctx context.Context, jobStatus *JobStatus, req *models.PrivacyRequest) {
	// Update status to running
	s.jobTracker.UpdateJobStatus(jobStatus.JobID, JobRunning, "")
	s.auditLogger.LogEvent("job_started", "Job processing started", jobStatus.JobID, jobStatus.RequestID, JobRunning, nil)

	// Send request to DP connector
	dpResp, err := s.dpService.VerifyWithDP(ctx, req)
	if err != nil {
		s.handleJobFailure(jobStatus, err)
		return
	}

	// Parse and validate response
	result, err := s.parseJobResult(dpResp)
	if err != nil {
		s.handleJobFailure(jobStatus, err)
		return
	}

	// Update job with result
	s.jobTracker.CompleteJob(jobStatus.JobID, result, "")
	s.auditLogger.LogEvent("job_completed", "Job completed successfully", jobStatus.JobID, jobStatus.RequestID, JobCompleted, map[string]string{
		"verified": fmt.Sprintf("%t", result.Verified),
		"confidence": fmt.Sprintf("%.2f", result.ConfidenceScore),
	})
}

// handleJobFailure handles job failures with retry logic
func (s *PullJobService) handleJobFailure(jobStatus *JobStatus, err error) {
	jobStatus.RetryCount++
	
	if jobStatus.RetryCount <= jobStatus.MaxRetries {
		// Retry the job
		s.auditLogger.LogEvent("job_retry", fmt.Sprintf("Job retry %d/%d", jobStatus.RetryCount, jobStatus.MaxRetries), 
			jobStatus.JobID, jobStatus.RequestID, JobPending, map[string]string{
				"retry_count": fmt.Sprintf("%d", jobStatus.RetryCount),
				"error":       err.Error(),
			})
		
		// Schedule retry with exponential backoff
		delay := time.Duration(jobStatus.RetryCount) * time.Second
		time.AfterFunc(delay, func() {
			// Re-process the job
			// Note: In a real implementation, you'd re-fetch the original request
			s.auditLogger.LogEvent("job_retry_started", "Job retry started", 
				jobStatus.JobID, jobStatus.RequestID, JobRunning, nil)
		})
	} else {
		// Max retries exceeded, mark as failed
		s.jobTracker.UpdateJobStatus(jobStatus.JobID, JobFailed, err.Error())
		s.auditLogger.LogEvent("job_failed", "Job failed after max retries", 
			jobStatus.JobID, jobStatus.RequestID, JobFailed, map[string]string{
				"error": err.Error(),
			})
	}
}

// parseJobResult parses and validates the job result
func (s *PullJobService) parseJobResult(dpResp *DPResponse) (*models.DPResponse, error) {
	// Validate response structure
	if dpResp.JobID == "" {
		return nil, fmt.Errorf("invalid response: missing job ID")
	}

	if dpResp.Status == "" {
		return nil, fmt.Errorf("invalid response: missing status")
	}

	// Convert DPResponse to models.DPResponse
	result := &models.DPResponse{
		Status:         dpResp.Status,
		ConfidenceScore: 0.0,
		DPID:          "dp-connector",
		Timestamp:     dpResp.Timestamp,
	}

	// Extract verification result if available
	if dpResp.VerificationResult != nil {
		result.Verified = dpResp.VerificationResult.Verified
		result.ConfidenceScore = dpResp.VerificationResult.Confidence
		result.Reason = dpResp.VerificationResult.Reason
	}

	// Validate confidence score
	if result.ConfidenceScore < 0.0 || result.ConfidenceScore > 1.0 {
		return nil, fmt.Errorf("invalid confidence score: %f", result.ConfidenceScore)
	}

	return result, nil
}

// GetJobStatus retrieves the status of a job
func (s *PullJobService) GetJobStatus(jobID string) (*JobStatus, error) {
	return s.jobTracker.GetJob(jobID)
}

// ListJobs lists all tracked jobs
func (s *PullJobService) ListJobs() []*JobStatus {
	return s.jobTracker.ListJobs()
}

// CleanupExpiredJobs removes expired jobs from tracking
func (s *PullJobService) CleanupExpiredJobs() {
	s.jobTracker.CleanupExpiredJobs()
}

// generateJobID generates a unique job ID
func (s *PullJobService) generateJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}

// TrackJob tracks a new job
func (jt *JobTracker) TrackJob(job *JobStatus) {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	jt.jobs[job.JobID] = job
}

// UpdateJobStatus updates the status of a job
func (jt *JobTracker) UpdateJobStatus(jobID string, status JobState, error string) {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	
	if job, exists := jt.jobs[jobID]; exists {
		job.Status = status
		job.UpdatedAt = time.Now()
		if error != "" {
			job.Error = error
		}
	}
}

// CompleteJob marks a job as completed with result
func (jt *JobTracker) CompleteJob(jobID string, result *models.DPResponse, error string) {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	
	if job, exists := jt.jobs[jobID]; exists {
		job.Status = JobCompleted
		job.UpdatedAt = time.Now()
		completedAt := time.Now()
		job.CompletedAt = &completedAt
		job.Result = result
		if error != "" {
			job.Error = error
		}
	}
}

// GetJob retrieves a job by ID
func (jt *JobTracker) GetJob(jobID string) (*JobStatus, error) {
	jt.mu.RLock()
	defer jt.mu.RUnlock()
	
	if job, exists := jt.jobs[jobID]; exists {
		return job, nil
	}
	return nil, fmt.Errorf("job not found: %s", jobID)
}

// ListJobs lists all tracked jobs
func (jt *JobTracker) ListJobs() []*JobStatus {
	jt.mu.RLock()
	defer jt.mu.RUnlock()
	
	jobs := make([]*JobStatus, 0, len(jt.jobs))
	for _, job := range jt.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// CleanupExpiredJobs removes jobs that are older than 24 hours
func (jt *JobTracker) CleanupExpiredJobs() {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	
	cutoff := time.Now().Add(-24 * time.Hour)
	for jobID, job := range jt.jobs {
		if job.UpdatedAt.Before(cutoff) {
			delete(jt.jobs, jobID)
		}
	}
}

// LogEvent logs a job audit event
func (jal *JobAuditLogger) LogEvent(eventType, description, jobID, requestID string, status JobState, metadata map[string]string) {
	jal.mu.Lock()
	defer jal.mu.Unlock()

	event := JobAuditEvent{
		Timestamp:   time.Now().Format(time.RFC3339),
		EventType:   eventType,
		JobID:       jobID,
		RequestID:   requestID,
		Description: description,
		Status:      status,
		Metadata:    metadata,
	}

	jal.events = append(jal.events, event)

	// Keep only the last 1000 events
	if len(jal.events) > 1000 {
		jal.events = jal.events[len(jal.events)-1000:]
	}
}

// GetAuditEvents returns job audit events
func (jal *JobAuditLogger) GetAuditEvents() []JobAuditEvent {
	jal.mu.Lock()
	defer jal.mu.Unlock()

	events := make([]JobAuditEvent, len(jal.events))
	copy(events, jal.events)
	return events
}

// GetJobStats returns job statistics
func (s *PullJobService) GetJobStats() map[string]interface{} {
	jobs := s.jobTracker.ListJobs()
	
	stats := map[string]interface{}{
		"total_jobs":     len(jobs),
		"pending_jobs":   0,
		"running_jobs":   0,
		"completed_jobs": 0,
		"failed_jobs":    0,
		"timeout_jobs":   0,
	}

	for _, job := range jobs {
		switch job.Status {
		case JobPending:
			stats["pending_jobs"] = stats["pending_jobs"].(int) + 1
		case JobRunning:
			stats["running_jobs"] = stats["running_jobs"].(int) + 1
		case JobCompleted:
			stats["completed_jobs"] = stats["completed_jobs"].(int) + 1
		case JobFailed:
			stats["failed_jobs"] = stats["failed_jobs"].(int) + 1
		case JobTimeout:
			stats["timeout_jobs"] = stats["timeout_jobs"].(int) + 1
		}
	}

	return stats
}

// HealthCheck checks if the pull-job service is healthy
func (s *PullJobService) HealthCheck(ctx context.Context) error {
	// Check job tracker
	jobs := s.jobTracker.ListJobs()
	if len(jobs) > 1000 {
		return fmt.Errorf("too many tracked jobs: %d", len(jobs))
	}

	// Check audit logger
	events := s.auditLogger.GetAuditEvents()
	if len(events) > 1000 {
		return fmt.Errorf("too many audit events: %d", len(events))
	}

	return nil
} 