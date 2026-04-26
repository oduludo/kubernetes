package wait

import (
	"os"
	"testing"

	"github.com/google/cel-go/cel"
)

func TestRunCelCommandOnNativeType(t *testing.T) {
	// While YAML would structure the Pod name as metadata.name, the Pod struct embeds ObjectMeta, which has the Name
	// field, leading to `pod.Name` on the native Go Pod type.
	u := createUnstructured(t, podYAML)
	val, err := runCelCommandOnNativeType(u, cel.DynType, "pod.Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Value() != "foo-b6699dcfb-rnv7t" {
		t.Fatalf("unexpected value: %v", val.Value())
	}
}

func TestRunCelCommandOnUnstructuredMapping(t *testing.T) {
	u := createUnstructured(t, podYAML)
	val, err := runCelCommandOnUnstructuredMapping(u, cel.DynType, "pod", "pod.metadata.name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Value() != "foo-b6699dcfb-rnv7t" {
		t.Fatalf("unexpected value: %v", val.Value())
	}
}

func TestRunCelCommandOnUnstructuredMapping_DeepNestedField(t *testing.T) {
	u := createUnstructured(t, podYAML)
	val, err := runCelCommandOnUnstructuredMapping(u, cel.DynType, "pod", "pod.spec.containers[0].resources.limits.cpu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Value() != "500m" {
		t.Fatalf("unexpected value: %v", val.Value())
	}
}

func TestCELWait_checkConditionMet(t *testing.T) {
	u := createUnstructured(t, podYAML)
	c := CELWait{
		query:  "pod.metadata.name == 'foo-b6699dcfb-rnv7t'",
		errOut: os.Stderr,
	}
	met, err := c.checkCondition(u)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if !met {
		t.Fatal("expected condition to be met")
	}
}

func TestCELWait_checkConditionNotMet(t *testing.T) {
	u := createUnstructured(t, podYAML)
	c := CELWait{
		query:  "pod.metadata.name == 'incorrect-name'",
		errOut: os.Stderr,
	}
	met, err := c.checkCondition(u)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if met {
		t.Fatal("expected condition not to be met")
	}
}
