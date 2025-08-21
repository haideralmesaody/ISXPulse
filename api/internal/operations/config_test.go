package operations_test

import (
	"testing"
	"time"

	"isxcli/internal/operations"
	"isxcli/internal/operations/testutil"
)

func TestConfigGetStepConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         *operations.Config
		stageID        string
		expectedConfig interface{}
		expectedOK     bool
	}{
		{
			name:           "nil StepConfigs map",
			config:         &operations.Config{},
			stageID:        "test-Step",
			expectedConfig: nil,
			expectedOK:     false,
		},
		{
			name: "empty StepConfigs map",
			config: &operations.Config{
				StepConfigs: make(map[string]interface{}),
			},
			stageID:        "test-Step",
			expectedConfig: nil,
			expectedOK:     false,
		},
		{
			name: "existing Step config",
			config: &operations.Config{
				StepConfigs: map[string]interface{}{
					"test-Step": map[string]interface{}{
						"enabled": true,
						"timeout": "30s",
					},
				},
			},
			stageID: "test-Step",
			expectedConfig: map[string]interface{}{
				"enabled": true,
				"timeout": "30s",
			},
			expectedOK: true,
		},
		{
			name: "non-existing Step config",
			config: &operations.Config{
				StepConfigs: map[string]interface{}{
					"other-Step": "config",
				},
			},
			stageID:        "test-Step",
			expectedConfig: nil,
			expectedOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, ok := tt.config.GetStepConfig(tt.stageID)
			
			if ok != tt.expectedOK {
				t.Errorf("GetStepConfig() ok = %v, want %v", ok, tt.expectedOK)
			}
			
			if tt.expectedOK {
				expectedMap := tt.expectedConfig.(map[string]interface{})
				actualMap := config.(map[string]interface{})
				
				for key, expectedValue := range expectedMap {
					if actualValue, exists := actualMap[key]; !exists || actualValue != expectedValue {
						t.Errorf("GetStepConfig() config[%s] = %v, want %v", key, actualValue, expectedValue)
					}
				}
			} else if config != nil {
				t.Errorf("GetStepConfig() config = %v, want nil", config)
			}
		})
	}
}

func TestConfigSetStepConfig(t *testing.T) {
	tests := []struct {
		name           string
		initialConfig  *operations.Config
		stageID        string
		configToSet    interface{}
		expectedConfig interface{}
	}{
		{
			name:          "set config on nil StepConfigs",
			initialConfig: &operations.Config{},
			stageID:       "test-Step",
			configToSet: map[string]interface{}{
				"enabled": true,
				"retries": 3,
			},
			expectedConfig: map[string]interface{}{
				"enabled": true,
				"retries": 3,
			},
		},
		{
			name: "set config on existing StepConfigs",
			initialConfig: &operations.Config{
				StepConfigs: map[string]interface{}{
					"existing-Step": "existing-config",
				},
			},
			stageID:     "test-Step",
			configToSet: "new-config",
			expectedConfig: "new-config",
		},
		{
			name: "overwrite existing Step config",
			initialConfig: &operations.Config{
				StepConfigs: map[string]interface{}{
					"test-Step": "old-config",
				},
			},
			stageID:        "test-Step",
			configToSet:    "new-config",
			expectedConfig: "new-config",
		},
		{
			name:          "set complex struct config",
			initialConfig: &operations.Config{},
			stageID:       "complex-Step",
			configToSet: struct {
				Enabled    bool
				MaxRetries int
				Timeout    time.Duration
			}{
				Enabled:    true,
				MaxRetries: 5,
				Timeout:    30 * time.Second,
			},
			expectedConfig: struct {
				Enabled    bool
				MaxRetries int
				Timeout    time.Duration
			}{
				Enabled:    true,
				MaxRetries: 5,
				Timeout:    30 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initialConfig.SetStepConfig(tt.stageID, tt.configToSet)
			
			// Verify StepConfigs map was created if it was nil
			if tt.initialConfig.StepConfigs == nil {
				t.Error("SetStepConfig() should create StepConfigs map if nil")
				return
			}
			
			// Verify the config was set correctly
			actualConfig, ok := tt.initialConfig.GetStepConfig(tt.stageID)
			if !ok {
				t.Errorf("SetStepConfig() failed to set config for Step %s", tt.stageID)
				return
			}
			
			// Handle map comparison separately since maps are not comparable
			if expectedMap, isMap := tt.expectedConfig.(map[string]interface{}); isMap {
				actualMap, ok := actualConfig.(map[string]interface{})
				if !ok {
					t.Errorf("SetStepConfig() actualConfig is not a map, got %T", actualConfig)
					return
				}
				for key, expectedValue := range expectedMap {
					if actualValue, exists := actualMap[key]; !exists || actualValue != expectedValue {
						t.Errorf("SetStepConfig() config[%s] = %v, want %v", key, actualValue, expectedValue)
					}
				}
			} else {
				testutil.AssertEqual(t, actualConfig, tt.expectedConfig)
			}
		})
	}
}

func TestConfigBuilderWithCheckpoints(t *testing.T) {
	tests := []struct {
		name           string
		enabled        bool
		dir            string
		expectedConfig *operations.Config
	}{
		{
			name:    "enable checkpoints with directory",
			enabled: true,
			dir:     "/tmp/checkpoints", 
			expectedConfig: &operations.Config{
				ExecutionMode:    operations.ExecutionModeSequential,
				EnableCheckpoints: true,
				CheckpointDir:    "/tmp/checkpoints",
				ContinueOnError:  false,
				MaxConcurrency:   1,
				RetryConfig: operations.RetryConfig{
					MaxAttempts:  1,
					InitialDelay: 1 * time.Second,
					MaxDelay:     30 * time.Second,
					Multiplier:   2.0,
				},
				StageTimeouts: make(map[string]time.Duration),
			},
		},
		{
			name:    "enable checkpoints without directory",
			enabled: true,
			dir:     "",
			expectedConfig: &operations.Config{
				ExecutionMode:    operations.ExecutionModeSequential,
				EnableCheckpoints: true,
				CheckpointDir:    "data/checkpoints", // Should keep default directory
				ContinueOnError:  false,
				MaxConcurrency:   1,
				RetryConfig: operations.RetryConfig{
					MaxAttempts:  1,
					InitialDelay: 1 * time.Second,
					MaxDelay:     30 * time.Second,
					Multiplier:   2.0,
				},
				StageTimeouts: make(map[string]time.Duration),
			},
		},
		{
			name:    "disable checkpoints with directory",
			enabled: false,
			dir:     "/tmp/checkpoints",
			expectedConfig: &operations.Config{
				ExecutionMode:    operations.ExecutionModeSequential,
				EnableCheckpoints: false,
				CheckpointDir:    "/tmp/checkpoints", // Should still be set
				ContinueOnError:  false,
				MaxConcurrency:   1,
				RetryConfig: operations.RetryConfig{
					MaxAttempts:  1,
					InitialDelay: 1 * time.Second,
					MaxDelay:     30 * time.Second,
					Multiplier:   2.0,
				},
				StageTimeouts: make(map[string]time.Duration),
			},
		},
		{
			name:    "disable checkpoints without directory",
			enabled: false,
			dir:     "",
			expectedConfig: &operations.Config{
				ExecutionMode:    operations.ExecutionModeSequential,
				EnableCheckpoints: false,
				CheckpointDir:    "data/checkpoints", // Should keep default directory
				ContinueOnError:  false,
				MaxConcurrency:   1,
				RetryConfig: operations.RetryConfig{
					MaxAttempts:  1,
					InitialDelay: 1 * time.Second,
					MaxDelay:     30 * time.Second,
					Multiplier:   2.0,
				},
				StageTimeouts: make(map[string]time.Duration),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := operations.NewConfigBuilder().
				WithCheckpoints(tt.enabled, tt.dir).
				Build()

			testutil.AssertEqual(t, config.EnableCheckpoints, tt.expectedConfig.EnableCheckpoints)
			testutil.AssertEqual(t, config.CheckpointDir, tt.expectedConfig.CheckpointDir)
			
			// Verify other default values remain unchanged
			testutil.AssertEqual(t, config.ExecutionMode, tt.expectedConfig.ExecutionMode)
			testutil.AssertEqual(t, config.ContinueOnError, tt.expectedConfig.ContinueOnError)
			testutil.AssertEqual(t, config.MaxConcurrency, tt.expectedConfig.MaxConcurrency)
		})
	}
}

func TestConfigBuilderWithStepConfig(t *testing.T) {
	tests := []struct {
		name         string
		stageID      string
		StepConfig  interface{}
		expectedOK   bool
	}{
		{
			name:    "set simple string config",
			stageID: "test-Step",
			StepConfig: "simple-config",
			expectedOK: true,
		},
		{
			name:    "set map config",
			stageID: "map-Step",
			StepConfig: map[string]interface{}{
				"enabled":     true,
				"max_retries": 3,
				"timeout":     "30s",
			},
			expectedOK: true,
		},
		{
			name:    "set struct config", 
			stageID: "struct-Step",
			StepConfig: struct {
				Name    string
				Enabled bool
				Count   int
			}{
				Name:    "test-struct",
				Enabled: true,
				Count:   42,
			},
			expectedOK: true,
		},
		{
			name:        "set nil config",
			stageID:     "nil-Step",
			StepConfig: nil,
			expectedOK:  true, // nil is a valid config value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := operations.NewConfigBuilder().
				WithStepConfig(tt.stageID, tt.StepConfig).
				Build()

			actualConfig, ok := config.GetStepConfig(tt.stageID)
			
			if ok != tt.expectedOK {
				t.Errorf("WithStepConfig() ok = %v, want %v", ok, tt.expectedOK)
			}
			
			if tt.expectedOK {
				// Handle map comparison separately since maps are not comparable
				if expectedMap, isMap := tt.StepConfig.(map[string]interface{}); isMap {
					actualMap, ok := actualConfig.(map[string]interface{})
					if !ok {
						t.Errorf("WithStepConfig() actualConfig is not a map, got %T", actualConfig)
						return
					}
					for key, expectedValue := range expectedMap {
						if actualValue, exists := actualMap[key]; !exists || actualValue != expectedValue {
							t.Errorf("WithStepConfig() config[%s] = %v, want %v", key, actualValue, expectedValue)
						}
					}
				} else {
					testutil.AssertEqual(t, actualConfig, tt.StepConfig)
				}
			}
		})
	}
}

func TestConfigBuilderChaining(t *testing.T) {
	// Test that all builder methods can be chained together
	config := operations.NewConfigBuilder().
		WithExecutionMode(operations.ExecutionModeParallel).
		WithStageTimeout("stage1", 30*time.Second).
		WithRetryConfig(operations.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 2 * time.Second,
			MaxDelay:     60 * time.Second,
			Multiplier:   2.5,
		}).
		WithContinueOnError(true).
		WithMaxConcurrency(5).
		WithCheckpoints(true, "/opt/checkpoints").
		WithStepConfig("custom-Step", map[string]interface{}{
			"feature_enabled": true,
			"batch_size":     100,
		}).
		Build()

	// Verify all configurations were applied
	testutil.AssertEqual(t, config.ExecutionMode, operations.ExecutionModeParallel)
	testutil.AssertEqual(t, config.ContinueOnError, true)
	testutil.AssertEqual(t, config.MaxConcurrency, 5)
	testutil.AssertEqual(t, config.EnableCheckpoints, true)
	testutil.AssertEqual(t, config.CheckpointDir, "/opt/checkpoints")
	
	timeout := config.GetStageTimeout("stage1")
	testutil.AssertEqual(t, timeout, 30*time.Second)
	
	testutil.AssertEqual(t, config.RetryConfig.MaxAttempts, 3)
	testutil.AssertEqual(t, config.RetryConfig.Multiplier, 2.5)
	
	StepConfig, ok := config.GetStepConfig("custom-Step")
	testutil.AssertEqual(t, ok, true)
	expectedStepConfig := map[string]interface{}{
		"feature_enabled": true,
		"batch_size":     100,
	}
	
	// Handle map comparison separately since maps are not comparable
	actualMap, ok := StepConfig.(map[string]interface{})
	if !ok {
		t.Errorf("StepConfig is not a map, got %T", StepConfig)
	} else {
		for key, expectedValue := range expectedStepConfig {
			if actualValue, exists := actualMap[key]; !exists || actualValue != expectedValue {
				t.Errorf("StepConfig[%s] = %v, want %v", key, actualValue, expectedValue)
			}
		}
	}
}

func TestConfigBuilderMultipleStepConfigs(t *testing.T) {
	config := operations.NewConfigBuilder().
		WithStepConfig("stage1", "config1").
		WithStepConfig("stage2", "config2").
		WithStepConfig("stage3", map[string]interface{}{
			"enabled": false,
		}).
		Build()

	// Verify all Step configs were set
	config1, ok1 := config.GetStepConfig("stage1")
	testutil.AssertEqual(t, ok1, true)
	testutil.AssertEqual(t, config1, "config1")

	config2, ok2 := config.GetStepConfig("stage2")
	testutil.AssertEqual(t, ok2, true) 
	testutil.AssertEqual(t, config2, "config2")

	config3, ok3 := config.GetStepConfig("stage3")
	testutil.AssertEqual(t, ok3, true)
	expectedConfig3 := map[string]interface{}{
		"enabled": false,
	}
	
	// Handle map comparison separately since maps are not comparable
	actualMap3, ok := config3.(map[string]interface{})
	if !ok {
		t.Errorf("config3 is not a map, got %T", config3)
	} else {
		for key, expectedValue := range expectedConfig3 {
			if actualValue, exists := actualMap3[key]; !exists || actualValue != expectedValue {
				t.Errorf("config3[%s] = %v, want %v", key, actualValue, expectedValue)
			}
		}
	}

	// Verify non-existent Step returns false
	_, ok4 := config.GetStepConfig("stage4")
	testutil.AssertEqual(t, ok4, false)
}