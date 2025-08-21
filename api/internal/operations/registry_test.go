package operations_test

import (
	"fmt"
	"sync"
	"testing"

	"isxcli/internal/operations"
	"isxcli/internal/operations/testutil"
)

func TestRegistry(t *testing.T) {
	registry := operations.NewRegistry()
	
	testutil.AssertNotNil(t, registry)
	testutil.AssertEqual(t, registry.Count(), 0)
	
	// List should return empty slice, not nil
	steps := registry.List()
	if steps == nil {
		t.Error("List() should return empty slice, not nil")
	}
	testutil.AssertEqual(t, len(steps), 0)
}

func TestRegistryRegister(t *testing.T) {
	registry := operations.NewRegistry()
	
	// Create and register steps
	stage1 := testutil.CreateSuccessfulStage("stage1", "Step 1")
	stage2 := testutil.CreateSuccessfulStage("stage2", "Step 2")
	stage3 := testutil.CreateSuccessfulStage("stage3", "Step 3")
	
	// Register steps
	testutil.AssertNoError(t, registry.Register(stage1))
	testutil.AssertNoError(t, registry.Register(stage2))
	testutil.AssertNoError(t, registry.Register(stage3))
	
	// Verify count
	testutil.AssertEqual(t, registry.Count(), 3)
	
	// Verify steps can be retrieved
	got1, err := registry.Get("stage1")
	testutil.AssertNoError(t, err)
	if got1 != stage1 {
		t.Error("Retrieved stage1 does not match registered Step")
	}
	
	// Verify order is maintained
	ids := registry.ListIDs()
	expectedOrder := []string{"stage1", "stage2", "stage3"}
	for i, id := range ids {
		if id != expectedOrder[i] {
			t.Errorf("Order[%d] = %s, want %s", i, id, expectedOrder[i])
		}
	}
}

func TestRegistryRegisterErrors(t *testing.T) {
	registry := operations.NewRegistry()
	
	// Test nil Step
	err := registry.Register(nil)
	testutil.AssertErrorContains(t, err, "nil Step")
	
	// Test empty ID
	emptyStage := &testutil.MockStage{
		IDValue:   "",
		NameValue: "Empty ID Step",
	}
	err = registry.Register(emptyStage)
	testutil.AssertErrorContains(t, err, "ID cannot be empty")
	
	// Test duplicate registration
	Step := testutil.CreateSuccessfulStage("dup", "Duplicate")
	testutil.AssertNoError(t, registry.Register(Step))
	
	err = registry.Register(Step)
	testutil.AssertErrorContains(t, err, "already registered")
}

func TestRegistryUnregister(t *testing.T) {
	registry := operations.NewRegistry()
	
	// Register steps
	stage1 := testutil.CreateSuccessfulStage("stage1", "Step 1")
	stage2 := testutil.CreateSuccessfulStage("stage2", "Step 2")
	stage3 := testutil.CreateSuccessfulStage("stage3", "Step 3")
	
	registry.Register(stage1)
	registry.Register(stage2)
	registry.Register(stage3)
	
	// Unregister stage2
	testutil.AssertNoError(t, registry.Unregister("stage2"))
	
	// Verify count
	testutil.AssertEqual(t, registry.Count(), 2)
	
	// Verify stage2 is gone
	_, err := registry.Get("stage2")
	testutil.AssertErrorContains(t, err, "not found")
	
	// Verify order is updated
	ids := registry.ListIDs()
	expectedOrder := []string{"stage1", "stage3"}
	for i, id := range ids {
		if id != expectedOrder[i] {
			t.Errorf("Order[%d] = %s, want %s", i, id, expectedOrder[i])
		}
	}
	
	// Test unregistering non-existent Step
	err = registry.Unregister("nonexistent")
	testutil.AssertErrorContains(t, err, "not found")
}

func TestRegistryGet(t *testing.T) {
	registry := operations.NewRegistry()
	
	Step := testutil.CreateSuccessfulStage("test", "Test Step")
	registry.Register(Step)
	
	// Test successful get
	got, err := registry.Get("test")
	testutil.AssertNoError(t, err)
	if got != Step {
		t.Error("Retrieved Step does not match registered Step")
	}
	
	// Test get non-existent
	_, err = registry.Get("nonexistent")
	testutil.AssertErrorContains(t, err, "not found")
}

func TestRegistryHas(t *testing.T) {
	registry := operations.NewRegistry()
	
	Step := testutil.CreateSuccessfulStage("test", "Test Step")
	registry.Register(Step)
	
	// Test existing Step
	if !registry.Has("test") {
		t.Error("Has() should return true for existing Step")
	}
	
	// Test non-existent Step
	if registry.Has("nonexistent") {
		t.Error("Has() should return false for non-existent Step")
	}
}

func TestRegistryList(t *testing.T) {
	registry := operations.NewRegistry()
	
	// Create steps
	steps := []operations.Step{
		testutil.CreateSuccessfulStage("s1", "Step 1"),
		testutil.CreateSuccessfulStage("s2", "Step 2"),
		testutil.CreateSuccessfulStage("s3", "Step 3"),
	}
	
	// Register in specific order
	for _, Step := range steps {
		registry.Register(Step)
	}
	
	// List should return in registration order
	listed := registry.List()
	if len(listed) != len(steps) {
		t.Errorf("List() returned %d steps, want %d", len(listed), len(steps))
	}
	
	for i, Step := range listed {
		if Step.ID() != steps[i].ID() {
			t.Errorf("List()[%d].ID = %s, want %s", i, Step.ID(), steps[i].ID())
		}
	}
}

func TestRegistryClear(t *testing.T) {
	registry := operations.NewRegistry()
	
	// Add some steps
	registry.Register(testutil.CreateSuccessfulStage("s1", "Step 1"))
	registry.Register(testutil.CreateSuccessfulStage("s2", "Step 2"))
	registry.Register(testutil.CreateSuccessfulStage("s3", "Step 3"))
	
	// Verify steps exist
	testutil.AssertEqual(t, registry.Count(), 3)
	
	// Clear
	registry.Clear()
	
	// Verify empty
	testutil.AssertEqual(t, registry.Count(), 0)
	testutil.AssertEqual(t, len(registry.List()), 0)
	testutil.AssertEqual(t, len(registry.ListIDs()), 0)
}

func TestRegistryGetDependencyOrder(t *testing.T) {
	tests := []struct {
		name          string
		steps        []testutil.MockStage
		expectedOrder []string
		wantErr       bool
		errContains   string
	}{
		{
			name: "No dependencies",
			steps: []testutil.MockStage{
				{IDValue: "a", NameValue: "A"},
				{IDValue: "b", NameValue: "B"},
				{IDValue: "c", NameValue: "C"},
			},
			expectedOrder: []string{"a", "b", "c"},
		},
		{
			name: "Linear dependencies",
			steps: []testutil.MockStage{
				{IDValue: "a", NameValue: "A"},
				{IDValue: "b", NameValue: "B", DependenciesValue: []string{"a"}},
				{IDValue: "c", NameValue: "C", DependenciesValue: []string{"b"}},
			},
			expectedOrder: []string{"a", "b", "c"},
		},
		{
			name: "Diamond dependencies",
			steps: []testutil.MockStage{
				{IDValue: "a", NameValue: "A"},
				{IDValue: "b", NameValue: "B", DependenciesValue: []string{"a"}},
				{IDValue: "c", NameValue: "C", DependenciesValue: []string{"a"}},
				{IDValue: "d", NameValue: "D", DependenciesValue: []string{"b", "c"}},
			},
			expectedOrder: []string{"a", "b", "c", "d"},
		},
		{
			name: "Circular dependency",
			steps: []testutil.MockStage{
				{IDValue: "a", NameValue: "A", DependenciesValue: []string{"b"}},
				{IDValue: "b", NameValue: "B", DependenciesValue: []string{"a"}},
			},
			wantErr:     true,
			errContains: "cycle detected",
		},
		{
			name: "Missing dependency",
			steps: []testutil.MockStage{
				{IDValue: "a", NameValue: "A"},
				{IDValue: "b", NameValue: "B", DependenciesValue: []string{"missing"}},
			},
			wantErr:     true,
			errContains: "non-existent Step",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := operations.NewRegistry()
			
			// Register steps
			for i := range tt.steps {
				registry.Register(&tt.steps[i])
			}
			
			// Get dependency order
			ordered, err := registry.GetDependencyOrder()
			
			if tt.wantErr {
				testutil.AssertErrorContains(t, err, tt.errContains)
				return
			}
			
			testutil.AssertNoError(t, err)
			
			// Verify order
			if len(ordered) != len(tt.expectedOrder) {
				t.Errorf("Ordered count = %d, want %d", len(ordered), len(tt.expectedOrder))
				return
			}
			
			// For diamond case, b and c can be in any order
			if tt.name == "Diamond dependencies" {
				// Just verify a is first and d is last
				if ordered[0].ID() != "a" {
					t.Error("First Step should be 'a'")
				}
				if ordered[3].ID() != "d" {
					t.Error("Last Step should be 'd'")
				}
			} else {
				// For other cases, verify exact order
				for i, Step := range ordered {
					if Step.ID() != tt.expectedOrder[i] {
						t.Errorf("Order[%d] = %s, want %s", i, Step.ID(), tt.expectedOrder[i])
					}
				}
			}
		})
	}
}

func TestRegistryValidateDependencies(t *testing.T) {
	registry := operations.NewRegistry()
	
	// Register steps with valid dependencies
	stageA := testutil.CreateSuccessfulStage("a", "A")
	stageB := testutil.CreateSuccessfulStage("b", "B", "a")
	stageC := testutil.CreateSuccessfulStage("c", "C", "a", "b")
	
	registry.Register(stageA)
	registry.Register(stageB)
	registry.Register(stageC)
	
	// Should validate successfully
	testutil.AssertNoError(t, registry.ValidateDependencies())
	
	// Add Step with missing dependency
	stageD := testutil.CreateSuccessfulStage("d", "D", "missing")
	registry.Register(stageD)
	
	// Should fail validation
	err := registry.ValidateDependencies()
	testutil.AssertErrorContains(t, err, "non-existent Step")
}

func TestRegistryGetDependents(t *testing.T) {
	registry := operations.NewRegistry()
	
	// Create dependency tree:
	// a -> b -> d
	//   -> c -> d
	stageA := testutil.CreateSuccessfulStage("a", "A")
	stageB := testutil.CreateSuccessfulStage("b", "B", "a")
	stageC := testutil.CreateSuccessfulStage("c", "C", "a")
	stageD := testutil.CreateSuccessfulStage("d", "D", "b", "c")
	
	registry.Register(stageA)
	registry.Register(stageB)
	registry.Register(stageC)
	registry.Register(stageD)
	
	// Get dependents of 'a'
	dependentsA := registry.GetDependents("a")
	if len(dependentsA) != 2 {
		t.Errorf("Dependents of 'a' = %d, want 2", len(dependentsA))
	}
	
	// Get dependents of 'b'
	dependentsB := registry.GetDependents("b")
	if len(dependentsB) != 1 {
		t.Errorf("Dependents of 'b' = %d, want 1", len(dependentsB))
	}
	
	// Get dependents of 'd' (should be none)
	dependentsD := registry.GetDependents("d")
	if len(dependentsD) != 0 {
		t.Errorf("Dependents of 'd' = %d, want 0", len(dependentsD))
	}
}

func TestRegistryClone(t *testing.T) {
	registry := operations.NewRegistry()
	
	// Add steps
	stage1 := testutil.CreateSuccessfulStage("s1", "Step 1")
	stage2 := testutil.CreateSuccessfulStage("s2", "Step 2")
	stage3 := testutil.CreateSuccessfulStage("s3", "Step 3")
	
	registry.Register(stage1)
	registry.Register(stage2)
	registry.Register(stage3)
	
	// Clone
	clone := registry.Clone()
	
	// Verify clone has same steps
	testutil.AssertEqual(t, clone.Count(), registry.Count())
	
	// Verify order is preserved
	originalIDs := registry.ListIDs()
	cloneIDs := clone.ListIDs()
	for i, id := range originalIDs {
		if cloneIDs[i] != id {
			t.Errorf("Clone order[%d] = %s, want %s", i, cloneIDs[i], id)
		}
	}
	
	// Verify modifications to clone don't affect original
	clone.Clear()
	testutil.AssertEqual(t, registry.Count(), 3)
	testutil.AssertEqual(t, clone.Count(), 0)
}

func TestRegistryConcurrency(t *testing.T) {
	registry := operations.NewRegistry()
	
	var wg sync.WaitGroup
	operations := 100
	
	// Concurrent registrations
	wg.Add(operations)
	for i := 0; i < operations; i++ {
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("Step%d", n)
			Step := testutil.CreateSuccessfulStage(id, id)
			registry.Register(Step)
		}(i)
	}
	
	// Concurrent reads
	wg.Add(operations)
	for i := 0; i < operations; i++ {
		go func(n int) {
			defer wg.Done()
			registry.List()
			registry.ListIDs()
			registry.Count()
			registry.Has(fmt.Sprintf("Step%d", n))
		}(i)
	}
	
	wg.Wait()
	
	// Verify all steps were registered
	testutil.AssertEqual(t, registry.Count(), operations)
	
	// Verify all steps can be retrieved
	for i := 0; i < operations; i++ {
		id := fmt.Sprintf("Step%d", i)
		if !registry.Has(id) {
			t.Errorf("Step %s not found", id)
		}
	}
}