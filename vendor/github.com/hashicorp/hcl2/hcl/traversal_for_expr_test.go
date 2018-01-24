package hcl

import (
	"testing"
)

type asTraversalSupported struct {
	staticExpr
	RootName string
}

type asTraversalNotSupported struct {
	staticExpr
}

type asTraversalDeclined struct {
	staticExpr
}

func (e asTraversalSupported) AsTraversal() Traversal {
	return Traversal{
		TraverseRoot{
			Name: e.RootName,
		},
	}
}

func (e asTraversalDeclined) AsTraversal() Traversal {
	return nil
}

func TestAbsTraversalForExpr(t *testing.T) {
	tests := []struct {
		Expr         Expression
		WantRootName string
	}{
		{
			asTraversalSupported{RootName: "foo"},
			"foo",
		},
		{
			asTraversalNotSupported{},
			"",
		},
		{
			asTraversalDeclined{},
			"",
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got, diags := AbsTraversalForExpr(test.Expr)
			switch {
			case got != nil:
				if test.WantRootName == "" {
					t.Fatalf("traversal was returned; want error")
				}
				if len(got) != 1 {
					t.Fatalf("wrong traversal length %d; want 1", len(got))
				}
				gotRoot, ok := got[0].(TraverseRoot)
				if !ok {
					t.Fatalf("first traversal step is %T; want hcl.TraverseRoot", got[0])
				}
				if gotRoot.Name != test.WantRootName {
					t.Errorf("wrong root name %q; want %q", gotRoot.Name, test.WantRootName)
				}
			default:
				if !diags.HasErrors() {
					t.Errorf("returned nil traversal without error diagnostics")
				}
				if test.WantRootName != "" {
					t.Errorf("traversal was not returned; want TraverseRoot(%q)", test.WantRootName)
				}
			}
		})
	}
}

func TestRelTraversalForExpr(t *testing.T) {
	tests := []struct {
		Expr          Expression
		WantFirstName string
	}{
		{
			asTraversalSupported{RootName: "foo"},
			"foo",
		},
		{
			asTraversalNotSupported{},
			"",
		},
		{
			asTraversalDeclined{},
			"",
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got, diags := RelTraversalForExpr(test.Expr)
			switch {
			case got != nil:
				if test.WantFirstName == "" {
					t.Fatalf("traversal was returned; want error")
				}
				if len(got) != 1 {
					t.Fatalf("wrong traversal length %d; want 1", len(got))
				}
				gotRoot, ok := got[0].(TraverseAttr)
				if !ok {
					t.Fatalf("first traversal step is %T; want hcl.TraverseAttr", got[0])
				}
				if gotRoot.Name != test.WantFirstName {
					t.Errorf("wrong root name %q; want %q", gotRoot.Name, test.WantFirstName)
				}
			default:
				if !diags.HasErrors() {
					t.Errorf("returned nil traversal without error diagnostics")
				}
				if test.WantFirstName != "" {
					t.Errorf("traversal was not returned; want TraverseAttr(%q)", test.WantFirstName)
				}
			}
		})
	}
}
