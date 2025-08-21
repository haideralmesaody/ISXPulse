package operations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPipelineManifest(t *testing.T) {
	t.Run("NewManifest", func(t *testing.T) {
		manifest := NewPipelineManifest("op-123", "2025-01-01", "2025-01-31")
		
		assert.NotNil(t, manifest)
		assert.Equal(t, "op-123", manifest.OperationID)
		assert.Equal(t, "2025-01-01", manifest.FromDate)
		assert.Equal(t, "2025-01-31", manifest.ToDate)
		assert.Equal(t, "pending", manifest.Status)
		assert.NotNil(t, manifest.AvailableData)
		assert.Empty(t, manifest.CompletedStages)
	})

	t.Run("AddData", func(t *testing.T) {
		manifest := NewPipelineManifest("op-123", "2025-01-01", "2025-01-31")
		
		// Add excel files data
		manifest.AddData("excel_files", &DataInfo{
			Type:      "excel_files",
			Location:  "data/downloads",
			FileCount: 10,
			Files:     []string{"file1.xls", "file2.xls"},
		})
		
		// Check data was added
		assert.True(t, manifest.HasData("excel_files"))
		data, exists := manifest.GetData("excel_files")
		assert.True(t, exists)
		assert.Equal(t, 10, data.FileCount)
		assert.Equal(t, 2, len(data.Files))
	})

	t.Run("RecordStageExecution", func(t *testing.T) {
		manifest := NewPipelineManifest("op-123", "2025-01-01", "2025-01-31")
		
		// Record stage start
		manifest.RecordStageStart("scraping", "Data Collection")
		assert.Equal(t, 1, len(manifest.CompletedStages))
		assert.Equal(t, "running", manifest.CompletedStages[0].Status)
		
		// Record stage completion
		manifest.RecordStageCompletion("scraping", []string{"excel_files"}, nil)
		assert.Equal(t, "completed", manifest.CompletedStages[0].Status)
		assert.Contains(t, manifest.CompletedStages[0].OutputData, "excel_files")
	})

	t.Run("IsStageCompleted", func(t *testing.T) {
		manifest := NewPipelineManifest("op-123", "2025-01-01", "2025-01-31")
		
		// Stage not started
		assert.False(t, manifest.IsStageCompleted("scraping"))
		
		// Stage started but not completed
		manifest.RecordStageStart("scraping", "Data Collection")
		assert.False(t, manifest.IsStageCompleted("scraping"))
		
		// Stage completed
		manifest.RecordStageCompletion("scraping", []string{"excel_files"}, nil)
		assert.True(t, manifest.IsStageCompleted("scraping"))
	})

	t.Run("GetProgress", func(t *testing.T) {
		manifest := NewPipelineManifest("op-123", "2025-01-01", "2025-01-31")
		
		// No stages completed
		assert.Equal(t, 0, manifest.GetProgress())
		
		// One stage completed (25% for 1 of 4 stages)
		manifest.RecordStageStart("scraping", "Data Collection")
		manifest.RecordStageCompletion("scraping", []string{"excel_files"}, nil)
		assert.Equal(t, 25, manifest.GetProgress())
		
		// Two stages completed (50% for 2 of 4 stages)
		manifest.RecordStageStart("processing", "Data Processing")
		manifest.RecordStageCompletion("processing", []string{"csv_files"}, nil)
		assert.Equal(t, 50, manifest.GetProgress())
	})
}

func TestStageCanRun(t *testing.T) {
	t.Run("ScrapingStage_CanAlwaysRun", func(t *testing.T) {
		stage := &ScrapingStage{
			BaseStage: NewBaseStage("scraping", "Data Collection", nil),
		}
		manifest := NewPipelineManifest("op-123", "2025-01-01", "2025-01-31")
		
		// Should always be able to run
		assert.True(t, stage.CanRun(manifest))
	})

	t.Run("ProcessingStage_RequiresExcelFiles", func(t *testing.T) {
		stage := &ProcessingStage{
			BaseStage: NewBaseStage("processing", "Data Processing", nil),
		}
		manifest := NewPipelineManifest("op-123", "2025-01-01", "2025-01-31")
		
		// Cannot run without excel files
		assert.False(t, stage.CanRun(manifest))
		
		// Add excel files
		manifest.AddData("excel_files", &DataInfo{
			Type:      "excel_files",
			Location:  "data/downloads",
			FileCount: 5,
		})
		
		// Now can run
		assert.True(t, stage.CanRun(manifest))
	})

	t.Run("IndicesStage_RequiresCSVFiles", func(t *testing.T) {
		stage := &IndicesStage{
			BaseStage: NewBaseStage("indices", "Index Extraction", nil),
		}
		manifest := NewPipelineManifest("op-123", "2025-01-01", "2025-01-31")
		
		// Cannot run without CSV files
		assert.False(t, stage.CanRun(manifest))
		
		// Add CSV files
		manifest.AddData("csv_files", &DataInfo{
			Type:      "csv_files",
			Location:  "data/reports",
			FileCount: 3,
		})
		
		// Now can run
		assert.True(t, stage.CanRun(manifest))
	})

	t.Run("LiquidityStage_RequiresIndexData", func(t *testing.T) {
		stage := &LiquidityStage{
			BaseStage: NewBaseStage("liquidity", "Liquidity Calculation", nil),
		}
		manifest := NewPipelineManifest("op-123", "2025-01-01", "2025-01-31")
		
		// Cannot run without index data
		assert.False(t, stage.CanRun(manifest))
		
		// Add index data
		manifest.AddData("index_data", &DataInfo{
			Type:      "index_data",
			Location:  "data/reports",
			FileCount: 2,
		})
		
		// Now can run
		assert.True(t, stage.CanRun(manifest))
	})
}

func TestDataRequirements(t *testing.T) {
	t.Run("ScrapingStage_RequirementsAndOutputs", func(t *testing.T) {
		stage := &ScrapingStage{}
		
		// No requirements
		requirements := stage.RequiredInputs()
		assert.Empty(t, requirements)
		
		// Produces excel files
		outputs := stage.ProducedOutputs()
		assert.Len(t, outputs, 1)
		assert.Equal(t, "excel_files", outputs[0].Type)
		assert.Equal(t, "data/downloads", outputs[0].Location)
		assert.Equal(t, "*.xls", outputs[0].Pattern)
	})

	t.Run("ProcessingStage_RequirementsAndOutputs", func(t *testing.T) {
		stage := &ProcessingStage{}
		
		// Requires excel files
		requirements := stage.RequiredInputs()
		assert.Len(t, requirements, 1)
		assert.Equal(t, "excel_files", requirements[0].Type)
		assert.Equal(t, 1, requirements[0].MinCount)
		assert.False(t, requirements[0].Optional)
		
		// Produces CSV files
		outputs := stage.ProducedOutputs()
		assert.Len(t, outputs, 1)
		assert.Equal(t, "csv_files", outputs[0].Type)
		assert.Equal(t, "data/reports", outputs[0].Location)
		assert.Equal(t, "*.csv", outputs[0].Pattern)
	})

	t.Run("IndicesStage_RequirementsAndOutputs", func(t *testing.T) {
		stage := &IndicesStage{}
		
		// Requires CSV files
		requirements := stage.RequiredInputs()
		assert.Len(t, requirements, 1)
		assert.Equal(t, "csv_files", requirements[0].Type)
		assert.Equal(t, 1, requirements[0].MinCount)
		
		// Produces index data
		outputs := stage.ProducedOutputs()
		assert.Len(t, outputs, 1)
		assert.Equal(t, "index_data", outputs[0].Type)
		assert.Equal(t, "ISX*.csv", outputs[0].Pattern)
	})

	t.Run("LiquidityStage_RequirementsAndOutputs", func(t *testing.T) {
		stage := &LiquidityStage{}
		
		// Requires index data
		requirements := stage.RequiredInputs()
		assert.Len(t, requirements, 1)
		assert.Equal(t, "index_data", requirements[0].Type)
		assert.Equal(t, 1, requirements[0].MinCount)
		
		// Produces analysis results
		outputs := stage.ProducedOutputs()
		assert.Len(t, outputs, 1)
		assert.Equal(t, "analysis_results", outputs[0].Type)
		assert.Equal(t, "ticker_*.csv", outputs[0].Pattern)
	})
}