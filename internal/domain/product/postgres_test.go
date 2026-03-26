package product

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// buildProductFilterConditions
// ---------------------------------------------------------------------------

func TestBuildProductFilterConditions_NoFilter(t *testing.T) {
	conds, args := buildProductFilterConditions(ProductFilter{}, 1)
	if len(conds) != 0 {
		t.Errorf("expected no conditions, got %d", len(conds))
	}
	if len(args) != 0 {
		t.Errorf("expected no args, got %d", len(args))
	}
}

func TestBuildProductFilterConditions_ActiveOnly(t *testing.T) {
	active := true
	conds, args := buildProductFilterConditions(ProductFilter{Active: &active}, 1)

	if len(conds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(conds))
	}
	if conds[0] != "p.active = $1" {
		t.Errorf("unexpected condition: %q", conds[0])
	}
	if len(args) != 1 || args[0] != true {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestBuildProductFilterConditions_CategoryID_RecursiveCTE(t *testing.T) {
	catID := uuid.New()
	conds, args := buildProductFilterConditions(ProductFilter{CategoryID: &catID}, 1)

	if len(conds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(conds))
	}

	sql := conds[0]

	// Must use a recursive CTE to include descendants.
	if !strings.Contains(sql, "WITH RECURSIVE cat_descendants") {
		t.Error("condition missing WITH RECURSIVE cat_descendants")
	}
	if !strings.Contains(sql, "c.parent_id = cd.id") {
		t.Error("condition missing parent_id join for recursion")
	}
	if !strings.Contains(sql, "$1") {
		t.Error("condition missing $1 placeholder")
	}

	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
	if args[0] != catID {
		t.Errorf("expected category UUID %v, got %v", catID, args[0])
	}
}

// When Active is set before CategoryID the placeholder index must advance.
func TestBuildProductFilterConditions_CategoryID_PlaceholderOffset(t *testing.T) {
	active := false
	catID := uuid.New()
	conds, args := buildProductFilterConditions(ProductFilter{Active: &active, CategoryID: &catID}, 1)

	if len(conds) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(conds))
	}

	// First condition: Active uses $1.
	if !strings.Contains(conds[0], "$1") {
		t.Errorf("active condition should use $1, got: %q", conds[0])
	}

	// Second condition: CategoryID must use $2 (offset by Active's arg).
	if !strings.Contains(conds[1], "$2") {
		t.Errorf("category condition should use $2, got: %q", conds[1])
	}

	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[1] != catID {
		t.Errorf("expected catID as second arg, got %v", args[1])
	}
}

func TestBuildProductFilterConditions_SearchOnly(t *testing.T) {
	conds, args := buildProductFilterConditions(ProductFilter{Search: "boots"}, 1)

	if len(conds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(conds))
	}
	if !strings.Contains(conds[0], "plainto_tsquery") {
		t.Error("search condition missing plainto_tsquery")
	}
	if !strings.Contains(conds[0], "$1") {
		t.Error("search condition missing $1 placeholder")
	}
	if len(args) != 1 || args[0] != "boots" {
		t.Errorf("unexpected search args: %v", args)
	}
}

func TestBuildProductFilterConditions_AllFilters(t *testing.T) {
	active := true
	catID := uuid.New()
	conds, args := buildProductFilterConditions(ProductFilter{
		Active:     &active,
		CategoryID: &catID,
		Search:     "jacket",
	}, 1)

	if len(conds) != 3 {
		t.Fatalf("expected 3 conditions, got %d", len(conds))
	}
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(args))
	}

	// Verify placeholder sequence: $1, $2, $3.
	if !strings.Contains(conds[0], "$1") {
		t.Errorf("first condition should use $1: %q", conds[0])
	}
	if !strings.Contains(conds[1], "$2") {
		t.Errorf("second condition should use $2: %q", conds[1])
	}
	if !strings.Contains(conds[2], "$3") {
		t.Errorf("third condition should use $3: %q", conds[2])
	}
}
