package flag

import (
	"encoding/json"
	"reflect"
	"testing"
)

// mkFlag is a boolean flag with on/off variants — the common demo case.
func mkBoolFlag() Flag {
	return Flag{
		ID:         "f1",
		Product:    ProductRef{Team: "alpha", Product: "shop"},
		Key:        "checkout-variant",
		Type:       Bool,
		Variations: map[string]any{"on": true, "off": false},
	}
}

func TestProject_KillSwitch_DisablesAndSkipsTargeting(t *testing.T) {
	f := mkBoolFlag()
	cfg := EnvConfig{FlagID: "f1", Enabled: false, DefaultVariant: "off",
		Rules: []Rule{{Conditions: []Condition{{Attribute: "tenant", Op: OpEq, Values: []string{"acme"}}}, Variant: "on"}}}

	got := Project([]Flag{f}, map[string]EnvConfig{"f1": cfg})
	ff := got.Flags["checkout-variant"]

	if ff.State != "DISABLED" {
		t.Fatalf("kill switch should DISABLE, got %q", ff.State)
	}
	if ff.Targeting != nil {
		t.Fatalf("disabled flag must not carry targeting, got %v", ff.Targeting)
	}
	if ff.DefaultVariant != "off" {
		t.Fatalf("default variant = %q, want off", ff.DefaultVariant)
	}
}

func TestProject_NoConfig_IsDisabledAtDeterministicDefault(t *testing.T) {
	got := Project([]Flag{mkBoolFlag()}, nil)
	ff := got.Flags["checkout-variant"]
	if ff.State != "DISABLED" {
		t.Fatalf("un-configured flag must be DISABLED, got %q", ff.State)
	}
	if ff.DefaultVariant != "off" { // lexicographically first of {off,on}
		t.Fatalf("deterministic default = %q, want off", ff.DefaultVariant)
	}
}

func TestProject_Rules_CompileToJsonLogicIfChain(t *testing.T) {
	f := mkBoolFlag()
	cfg := EnvConfig{FlagID: "f1", Enabled: true, DefaultVariant: "off", Rules: []Rule{
		{Conditions: []Condition{{Attribute: "tenant", Op: OpEq, Values: []string{"acme"}}}, Variant: "on"},
	}}

	ff := Project([]Flag{f}, map[string]EnvConfig{"f1": cfg}).Flags["checkout-variant"]
	if ff.State != "ENABLED" {
		t.Fatalf("state = %q, want ENABLED", ff.State)
	}
	want := map[string]any{"if": []any{
		map[string]any{"==": []any{map[string]any{"var": "tenant"}, any("acme")}},
		"on",
	}}
	assertJSONEqual(t, want, ff.Targeting)
}

func TestProject_MultiConditionRule_IsAnded(t *testing.T) {
	f := mkBoolFlag()
	cfg := EnvConfig{FlagID: "f1", Enabled: true, Rules: []Rule{{
		Conditions: []Condition{
			{Attribute: "country", Op: OpIn, Values: []string{"US", "CA"}},
			{Attribute: "email", Op: OpEndsWith, Values: []string{"@acme.com"}},
		},
		Variant: "on",
	}}}
	ff := Project([]Flag{f}, map[string]EnvConfig{"f1": cfg}).Flags["checkout-variant"]
	want := map[string]any{"if": []any{
		map[string]any{"and": []any{
			map[string]any{"in": []any{map[string]any{"var": "country"}, []any{"US", "CA"}}},
			map[string]any{"ends_with": []any{map[string]any{"var": "email"}, any("@acme.com")}},
		}},
		"on",
	}}
	assertJSONEqual(t, want, ff.Targeting)
}

func TestProject_Rollout_CompilesToFractionalOnTargetingKey(t *testing.T) {
	f := mkBoolFlag()
	cfg := EnvConfig{FlagID: "f1", Enabled: true, Rules: []Rule{{
		Rollout: []RolloutBucket{{Variant: "on", Weight: 10}, {Variant: "off", Weight: 90}},
	}}}
	ff := Project([]Flag{f}, map[string]EnvConfig{"f1": cfg}).Flags["checkout-variant"]
	want := map[string]any{"if": []any{
		true, // unconditional rule
		map[string]any{"fractional": []any{
			map[string]any{"var": "targetingKey"},
			[]any{"on", 10},
			[]any{"off", 90},
		}},
	}}
	assertJSONEqual(t, want, ff.Targeting)
}

// assertJSONEqual compares two values by their JSON encoding — robust to []any vs concrete slice types
// and the any() wrappers, which is what matters for what flagd receives on the wire.
func assertJSONEqual(t *testing.T, want, got any) {
	t.Helper()
	wb, _ := json.Marshal(want)
	gb, _ := json.Marshal(got)
	var wm, gm any
	_ = json.Unmarshal(wb, &wm)
	_ = json.Unmarshal(gb, &gm)
	if !reflect.DeepEqual(wm, gm) {
		t.Fatalf("targeting mismatch:\n want %s\n got  %s", wb, gb)
	}
}
