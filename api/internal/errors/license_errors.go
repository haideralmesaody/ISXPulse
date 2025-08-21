package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"
)

// License-specific errors (using errors package for sentinel errors)
var (
	ErrLicenseExpired          = errors.New("license expired")
	ErrLicenseNotActivated     = errors.New("license not activated")
	ErrInvalidLicenseKey       = errors.New("invalid license key")
	ErrInvalidLicenseFormat    = errors.New("invalid license key format")
	ErrRateLimited            = errors.New("rate limited")
	ErrNetworkError           = errors.New("network error")
	ErrActivationFailed       = errors.New("activation failed")
	ErrLicenseValidationFailed = errors.New("license validation failed")
	ErrLicenseAlreadyActivated = errors.New("license already activated")
	
	// Reactivation-specific errors
	ErrLicenseReactivated           = errors.New("license reactivated")
	ErrReactivationLimitExceeded    = errors.New("reactivation limit exceeded")
	ErrAlreadyActivatedOnDevice     = errors.New("already activated on different device")
)

// LicenseActivationDetails provides additional context for license errors
type LicenseActivationDetails struct {
	ActivationDate   *time.Time `json:"activation_date,omitempty"`
	ExpiryDate       *time.Time `json:"expiry_date,omitempty"`
	DeviceInfo       string     `json:"device_info,omitempty"`
	CurrentStatus    string     `json:"current_status,omitempty"`
	DaysRemaining    int        `json:"days_remaining,omitempty"`
	SupportEmail     string     `json:"support_email,omitempty"`
	CanRecover       bool       `json:"can_recover,omitempty"`
	RecoveryDeadline *time.Time `json:"recovery_deadline,omitempty"`
	
	// Reactivation-specific fields
	ReactivationCount      int     `json:"reactivation_count,omitempty"`
	MaxReactivations       int     `json:"max_reactivations,omitempty"`
	SimilarityScore        float64 `json:"similarity_score,omitempty"`
	PreviousDeviceInfo     string  `json:"previous_device_info,omitempty"`
	ReactivationTimestamp  *time.Time `json:"reactivation_timestamp,omitempty"`
}

// ProblemDetails implements RFC 7807 Problem Details for HTTP APIs
type ProblemDetails struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
	
	// Additional fields for extensibility
	Extensions map[string]interface{} `json:"-"`
}

// Render implements the render.Renderer interface
func (pd *ProblemDetails) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, pd.Status)
	return nil
}

// MarshalJSON custom marshaler to include extensions
func (pd *ProblemDetails) MarshalJSON() ([]byte, error) {
	type Alias ProblemDetails
	data := make(map[string]interface{})
	
	// Add standard fields
	data["type"] = pd.Type
	data["title"] = pd.Title
	data["status"] = pd.Status
	
	if pd.Detail != "" {
		data["detail"] = pd.Detail
	}
	if pd.Instance != "" {
		data["instance"] = pd.Instance
	}
	
	// Add extensions
	for k, v := range pd.Extensions {
		data[k] = v
	}
	
	// Use standard JSON marshaling
	return json.Marshal(data)
}

// NewProblemDetails creates a new RFC 7807 compliant error
func NewProblemDetails(status int, problemType, title, detail, instance string) *ProblemDetails {
	return &ProblemDetails{
		Type:     problemType,
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: instance,
		Extensions: make(map[string]interface{}),
	}
}

// WithExtension adds an extension field to the problem details
func (pd *ProblemDetails) WithExtension(key string, value interface{}) *ProblemDetails {
	pd.Extensions[key] = value
	return pd
}

// NewLicenseAlreadyActivatedError creates an enhanced error for already activated licenses
func NewLicenseAlreadyActivatedError(details *LicenseActivationDetails, traceID string) *ProblemDetails {
	problem := NewProblemDetails(
		http.StatusConflict,
		"/errors/license-already-activated",
		"License Already Activated",
		"This license has already been activated on another device. To transfer it to this device, please contact support.",
		fmt.Sprintf("/api/license/activate#%s", traceID),
	)
	
	problem.WithExtension("error_type", "already_activated").
		WithExtension("trace_id", traceID).
		WithExtension("support_email", "support@isxpulse.com").
		WithExtension("transfer_info", "Contact support with your license key and proof of purchase to transfer this license.")
	
	if details != nil {
		if details.ActivationDate != nil {
			problem.WithExtension("original_activation_date", details.ActivationDate.Format("2006-01-02T15:04:05Z"))
		}
		if details.ExpiryDate != nil {
			problem.WithExtension("expiry_date", details.ExpiryDate.Format("2006-01-02T15:04:05Z"))
		}
		if details.DeviceInfo != "" {
			problem.WithExtension("registered_device", details.DeviceInfo)
		}
		if details.CurrentStatus != "" {
			problem.WithExtension("current_status", details.CurrentStatus)
		}
		problem.WithExtension("can_recover", details.CanRecover)
		if details.RecoveryDeadline != nil {
			problem.WithExtension("recovery_deadline", details.RecoveryDeadline.Format("2006-01-02T15:04:05Z"))
		}
	}
	
	return problem
}

// NewLicenseReactivatedResponse creates a success response for license reactivation
func NewLicenseReactivatedResponse(details *LicenseActivationDetails, traceID string) *ProblemDetails {
	problem := NewProblemDetails(
		http.StatusOK,
		"/success/license-reactivated",
		"License Successfully Reactivated",
		"Your license has been successfully reactivated on this device.",
		fmt.Sprintf("/api/license/activate#%s", traceID),
	)
	
	problem.WithExtension("success_type", "reactivated").
		WithExtension("trace_id", traceID).
		WithExtension("message", "License has been reactivated for this device")
	
	if details != nil {
		if details.ReactivationCount > 0 {
			problem.WithExtension("reactivation_count", details.ReactivationCount)
		}
		if details.MaxReactivations > 0 {
			problem.WithExtension("max_reactivations", details.MaxReactivations)
			problem.WithExtension("remaining_reactivations", details.MaxReactivations-details.ReactivationCount)
		}
		if details.SimilarityScore > 0 {
			problem.WithExtension("device_similarity_score", details.SimilarityScore)
		}
		if details.ReactivationTimestamp != nil {
			problem.WithExtension("reactivation_timestamp", details.ReactivationTimestamp.Format("2006-01-02T15:04:05Z"))
		}
		if details.ExpiryDate != nil {
			problem.WithExtension("expiry_date", details.ExpiryDate.Format("2006-01-02T15:04:05Z"))
		}
	}
	
	return problem
}

// NewReactivationLimitExceededError creates an error for when reactivation limit is reached
func NewReactivationLimitExceededError(details *LicenseActivationDetails, traceID string) *ProblemDetails {
	problem := NewProblemDetails(
		http.StatusConflict,
		"/errors/reactivation-limit-exceeded",
		"Reactivation Limit Exceeded",
		"This license has reached its maximum number of device reactivations. Please contact support for assistance.",
		fmt.Sprintf("/api/license/activate#%s", traceID),
	)
	
	problem.WithExtension("error_type", "reactivation_limit_exceeded").
		WithExtension("trace_id", traceID).
		WithExtension("support_email", "support@isxpulse.com").
		WithExtension("support_info", "Contact support with your license key to increase reactivation limit or transfer to a new device.")
	
	if details != nil {
		if details.ReactivationCount > 0 {
			problem.WithExtension("current_reactivations", details.ReactivationCount)
		}
		if details.MaxReactivations > 0 {
			problem.WithExtension("max_reactivations", details.MaxReactivations)
		}
		if details.ActivationDate != nil {
			problem.WithExtension("original_activation_date", details.ActivationDate.Format("2006-01-02T15:04:05Z"))
		}
		if details.ExpiryDate != nil {
			problem.WithExtension("expiry_date", details.ExpiryDate.Format("2006-01-02T15:04:05Z"))
		}
	}
	
	return problem
}

// NewAlreadyActivatedOnDeviceError creates an error for when license is already active on a different device
func NewAlreadyActivatedOnDeviceError(details *LicenseActivationDetails, traceID string) *ProblemDetails {
	problem := NewProblemDetails(
		http.StatusConflict,
		"/errors/already-activated-different-device",
		"License Already Active on Different Device",
		"This license is currently active on a different device. You may be able to reactivate it depending on device similarity and reactivation limits.",
		fmt.Sprintf("/api/license/activate#%s", traceID),
	)
	
	problem.WithExtension("error_type", "already_activated_different_device").
		WithExtension("trace_id", traceID).
		WithExtension("support_email", "support@isxpulse.com")
	
	if details != nil {
		if details.PreviousDeviceInfo != "" {
			problem.WithExtension("previous_device", details.PreviousDeviceInfo)
		}
		if details.SimilarityScore > 0 {
			problem.WithExtension("device_similarity_score", details.SimilarityScore)
		}
		if details.ReactivationCount > 0 && details.MaxReactivations > 0 {
			remaining := details.MaxReactivations - details.ReactivationCount
			problem.WithExtension("reactivations_remaining", remaining)
			if remaining > 0 {
				problem.WithExtension("can_reactivate", true)
				problem.WithExtension("reactivation_info", "This license can still be reactivated on this device.")
			} else {
				problem.WithExtension("can_reactivate", false)
				problem.WithExtension("reactivation_info", "Maximum reactivations reached. Contact support for assistance.")
			}
		}
		if details.ActivationDate != nil {
			problem.WithExtension("activation_date", details.ActivationDate.Format("2006-01-02T15:04:05Z"))
		}
		if details.ExpiryDate != nil {
			problem.WithExtension("expiry_date", details.ExpiryDate.Format("2006-01-02T15:04:05Z"))
		}
	}
	
	return problem
}

// MapLicenseError maps domain errors to HTTP problem details
func MapLicenseError(err error, traceID string) render.Renderer {
	instance := fmt.Sprintf("/api/license#trace-%s", traceID)
	
	// Check if it's an APIError from errors.go
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.ErrorCode == "LICENSE_NOT_FOUND" {
			return NewProblemDetails(
				http.StatusNotFound,
				"/errors/license-not-found",
				"License Not Found",
				"No license file found in the system. Please activate a license.",
				instance,
			).WithExtension("trace_id", traceID).
				WithExtension("error_code", "LICENSE_NOT_FOUND")
		}
	}
	
	switch {
	case errors.Is(err, ErrLicenseReactivated):
		return NewLicenseReactivatedResponse(nil, traceID)
	case errors.Is(err, ErrReactivationLimitExceeded):
		return NewReactivationLimitExceededError(nil, traceID)
	case errors.Is(err, ErrAlreadyActivatedOnDevice):
		return NewAlreadyActivatedOnDeviceError(nil, traceID)
	case errors.Is(err, ErrLicenseAlreadyActivated):
		return NewLicenseAlreadyActivatedError(nil, traceID)
	case errors.Is(err, ErrLicenseExpired):
		return NewProblemDetails(
			http.StatusForbidden,
			"/errors/license-expired",
			"License Expired",
			"Your license has expired. Please renew to continue.",
			instance,
		).WithExtension("trace_id", traceID).
			WithExtension("error_code", "LICENSE_EXPIRED")
			
			
	case errors.Is(err, ErrLicenseNotActivated):
		return NewProblemDetails(
			http.StatusPreconditionRequired,
			"/errors/license-not-activated",
			"License Not Activated",
			"No license has been activated. Please activate a license to continue.",
			instance,
		).WithExtension("trace_id", traceID).
			WithExtension("error_code", "LICENSE_NOT_ACTIVATED")
			
	case errors.Is(err, ErrInvalidLicenseKey):
		return NewProblemDetails(
			http.StatusBadRequest,
			"/errors/invalid-license-key",
			"Invalid License Key",
			"The provided license key is invalid or malformed.",
			instance,
		).WithExtension("trace_id", traceID).
			WithExtension("error_code", "INVALID_LICENSE_KEY")
			
	case errors.Is(err, ErrInvalidLicenseFormat):
		return NewProblemDetails(
			http.StatusBadRequest,
			"/errors/invalid-license-format",
			"Invalid License Format",
			"License key must be in format: ISX1Y-XXXXX-XXXXX-XXXXX-XXXXX",
			instance,
		).WithExtension("trace_id", traceID).
			WithExtension("error_code", "INVALID_LICENSE_FORMAT").
			WithExtension("expected_format", "ISX1Y-XXXXX-XXXXX-XXXXX-XXXXX")
			
	case errors.Is(err, ErrActivationFailed):
		return NewProblemDetails(
			http.StatusUnprocessableEntity,
			"/errors/activation-failed",
			"License Activation Failed",
			"Unable to activate the license. Please verify the key and try again.",
			instance,
		).WithExtension("trace_id", traceID).
			WithExtension("error_code", "ACTIVATION_FAILED")
			
	case errors.Is(err, ErrValidationFailed):
		return NewProblemDetails(
			http.StatusInternalServerError,
			"/errors/validation-failed",
			"License Validation Failed",
			"Unable to validate license status. Please try again later.",
			instance,
		).WithExtension("trace_id", traceID).
			WithExtension("error_code", "VALIDATION_FAILED")
			
	case errors.Is(err, ErrRateLimited):
		return NewProblemDetails(
			http.StatusTooManyRequests,
			"/errors/rate-limited",
			"Too Many Requests",
			"Too many activation attempts. Please try again later.",
			instance,
		).WithExtension("trace_id", traceID).
			WithExtension("error_code", "RATE_LIMITED").
			WithExtension("retry_after", 900) // 15 minutes
			
	case errors.Is(err, ErrNetworkError):
		return NewProblemDetails(
			http.StatusServiceUnavailable,
			"/errors/network-error",
			"Network Error",
			"Unable to connect to license server. Please check your connection.",
			instance,
		).WithExtension("trace_id", traceID).
			WithExtension("error_code", "NETWORK_ERROR")
			
	default:
		// Generic error
		return NewProblemDetails(
			http.StatusInternalServerError,
			"/errors/internal-error",
			"Internal Server Error",
			"An unexpected error occurred while processing your request.",
			instance,
		).WithExtension("trace_id", traceID).
			WithExtension("error_code", "INTERNAL_ERROR")
	}
}