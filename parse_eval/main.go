package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/y0ssar1an/q"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

func main() {
	hcl2TestSchema()
}

var buf []byte = []byte(`
suitename = "suite1"

testcase {
  casename = "case1"

  step {
    stepname = "step1"
  }

  step {
    stepname = "post-trailer"
    after = ["trailer"]
  }

  step {
    id = "trailer"
    stepname = "trailer"
    after = ["1", "s2", "4"]
  }

  step {
    id = "s2"
    stepname = "step2"
    before = ["1"]
  }

  step {
    stepname = "step3"
    after = ["s2"]
  }

  step {
    id = "pre-trailer"
    stepname = "pre-trailer"
    before = ["trailer"]
  }

  fixture {
    fixturename = "fixname1"
    some_rando = "blah ${upper(foo)} ${baz}"
  }
}

testcase {
  casename = "case2"

  step {
    stepname = "case2.step1"
  }
}
`)

type TestStep struct {
	Name    string
	StepNum uint64
	Config  hcl.Body
	id      string

	caseNode  graph.Node
	runBefore []string
	runAfter  []string
}

type TestCase struct {
	Name      string
	Enabled   bool
	TestSteps []*TestStep
	Fixtures  []*TestCaseFixture
	StepMap   map[string]*TestStep

	stepDepGraph    *simple.DirectedGraph
	stepDepRoot     graph.Node
	stepDepGraphMap map[graph.Node]*TestStep
}

type TestCaseFixture struct {
	Name   string
	Config hcl.Body
}

type TestSuite struct {
	Name      string
	TestCases []*TestCase
}

func hcl2TestSchema() {
	type rawTestStep struct {
		Name      string    `hcl:"stepname"`
		ID        *string   `hcl:"id,attr"`
		RunBefore *[]string `hcl:"before,attr"`
		RunAfter  *[]string `hcl:"after,attr"`
		Config    hcl.Body  `hcl:",remain"`
	}

	type rawTestCaseFixture struct {
		Name   string   `hcl:"fixturename,attr"`
		Config hcl.Body `hcl:",remain"`
	}

	type rawTestCase struct {
		Name      string                `hcl:"casename,attr"`
		Enabled   *bool                 `hcl:"enabled,attr"`
		TestSteps []rawTestStep         `hcl:"step,block"`
		Fixtures  []*rawTestCaseFixture `hcl:"fixture,block"`
	}

	type rawTestSuite struct {
		Name      string        `hcl:"suitename,attr"`
		TestCases []rawTestCase `hcl:"testcase,block"`
	}

	p := hclparse.NewParser()
	f, err := p.ParseHCL(buf, "stdin.hcl")
	if err != nil {
		panic(fmt.Sprintf("bad parse: %v", err))
	}

	ctx := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"foo": cty.StringVal("bar"),
			"baz": cty.NumberIntVal(5),
		},
		Functions: map[string]function.Function{
			"upper": stdlib.UpperFunc,
		},
	}

	{
		rts := rawTestSuite{}
		diags := gohcl.DecodeBody(f.Body, nil, &rts)
		if diags.HasErrors() {
			panic(fmt.Sprintf("bad decode: %v\n%v", diags, rts))
		}

		ts := TestSuite{
			Name: rts.Name,
		}

		ts.TestCases = make([]*TestCase, len(rts.TestCases))
		for i, rtc := range rts.TestCases {
			tc := &TestCase{
				Name:            rtc.Name,
				Fixtures:        []*TestCaseFixture{},
				TestSteps:       make([]*TestStep, 0, len(rtc.TestSteps)),
				StepMap:         make(map[string]*TestStep, len(rtc.TestSteps)),
				stepDepGraph:    simple.NewDirectedGraph(),
				stepDepGraphMap: make(map[graph.Node]*TestStep, len(rtc.TestSteps)),
			}
			tc.stepDepRoot = tc.stepDepGraph.NewNode()

			if rtc.Enabled == nil || *rtc.Enabled == true {
				tc.Enabled = true
			} else {
				tc.Enabled = false
			}

			tc.Fixtures = make([]*TestCaseFixture, 0, len(rtc.Fixtures))
			for _, rawFixture := range rtc.Fixtures {
				fixture := &TestCaseFixture{
					Name:   rawFixture.Name,
					Config: rawFixture.Config,
				}

				attrs, d := rawFixture.Config.JustAttributes()
				if d != nil {
					panic(fmt.Sprintf("%+v", d))
				}

				for k, v := range attrs {
					vty, diag := v.Expr.Value(ctx)
					if diag != nil {
						panic(fmt.Sprintf("%+v", diag))
					}
					q.Q("k", k, "v", vty.AsString())

					spew.Printf("fixture: suite=%q case=%q fixture.name=%q attr=%q value.name=%q value=%q\n", ts.Name, tc.Name, fixture.Name, k, v.Name, vty.AsString())
				}

				tc.Fixtures = append(tc.Fixtures, fixture)
			}

			tc.TestSteps = make([]*TestStep, 0, len(rtc.TestSteps))
			for i, rawStep := range rtc.TestSteps {
				stepNum := uint64(i) + 1
				step := &TestStep{
					Name:    rawStep.Name,
					Config:  rawStep.Config,
					StepNum: stepNum,

					caseNode: tc.stepDepGraph.NewNode(),
				}
				if rawStep.RunBefore != nil {
					step.runBefore = *rawStep.RunBefore
				}
				if rawStep.RunAfter != nil {
					step.runAfter = *rawStep.RunAfter
				}

				if rawStep.ID == nil || strings.TrimSpace(*rawStep.ID) == "" {
					step.id = strconv.FormatUint(stepNum, 10)
				} else {
					step.id = *rawStep.ID
				}

				attrs, d := step.Config.JustAttributes()
				if d != nil {
					panic(fmt.Sprintf("%+v", d))
				}

				for k, v := range attrs {
					vty, diag := v.Expr.Value(ctx)
					if diag != nil {
						panic(fmt.Sprintf("%+v", diag))
					}
					q.Q("k", k, "v", vty.GoString())
					spew.Printf("step: suite=%q case=%q step.id=%q attr=%q value.name=%q value=%q\n", ts.Name, tc.Name, step.id, k, v.Name, vty.GoString())
				}

				tc.StepMap[step.id] = step
				tc.TestSteps = append(tc.TestSteps, step)
				tc.stepDepGraph.AddNode(step.caseNode)
				tc.stepDepGraphMap[step.caseNode] = step
			}

			unconnectedSteps := make(map[graph.Node]*TestStep, len(tc.TestSteps))
			for _, step := range tc.TestSteps {
				unconnectedSteps[step.caseNode] = step
			}

			stepConnected := func(step *TestStep) {
				delete(unconnectedSteps, step.caseNode)
			}

			// Register dependencies
			for _, step := range tc.TestSteps {
				for _, before := range step.runBefore {
					s, found := tc.StepMap[before]
					if !found {
						fmt.Printf("step.id=%q's before dependency %q not found", step.id, before)
						continue
					}

					e := tc.stepDepGraph.NewEdge(step.caseNode, s.caseNode)
					tc.stepDepGraph.SetEdge(e)
					stepConnected(step)
					stepConnected(s)
				}

				for _, after := range step.runAfter {
					s, found := tc.StepMap[after]
					if !found {
						fmt.Printf("step.id=%q's after dependency %q not found", step.id, after)
						continue
					}

					e := tc.stepDepGraph.NewEdge(s.caseNode, step.caseNode)
					tc.stepDepGraph.SetEdge(e)
					stepConnected(step)
					stepConnected(s)
				}
			}

			// Ensure every node is part of the graph, otherwise add it to the nop
			// root node.
			rootID := tc.stepDepGraph.NewNode()
			tc.stepDepGraph.AddNode(rootID)
			tc.stepDepGraphMap[rootID] = nil
			for _, step := range unconnectedSteps {
				e := tc.stepDepGraph.NewEdge(rootID, step.caseNode)
				tc.stepDepGraph.SetEdge(e)
			}

			for k, v := range tc.StepMap {
				spew.Printf("step map: suite=%q case=%q step.id=%q step.name=%q\n", ts.Name, tc.Name, k, v.Name)
			}

			nodeCycles := topo.DirectedCyclesIn(tc.stepDepGraph)
			if len(nodeCycles) > 0 {
				for _, cycle := range nodeCycles {
					fmt.Printf("ERROR: cycle found between nodes: %+v\n", cycle)
				}

				fmt.Printf("WARNING: Skipping test case %q\n", tc.Name)
				continue
			}

			// orderedNodes, err := topo.SortStabilized(tc.stepDepGraph, nil)
			orderedNodes, err := topo.Sort(tc.stepDepGraph)
			if err != nil {
				panic(fmt.Sprintf("bad: %v", err))
			}

			if len(orderedNodes) < 1 {
				panic(fmt.Sprintf("should have more than 0 ordered nodes"))
				continue
			}
			fmt.Printf("%d nodes\n", len(orderedNodes)-1) // omit the root node

			for _, stepDepID := range orderedNodes {
				s, found := tc.stepDepGraphMap[stepDepID]
				if !found {
					panic(fmt.Sprintf("bad: node id not found %v in case %q", stepDepID, tc.Name))
				}

				if s == nil {
					// root node
					continue
				}

				fmt.Printf("executing suite=%q case=%q step(id=%q, name=%q)\n", ts.Name, tc.Name, s.id, s.Name)
			}

			ts.TestCases[i] = tc
		}
	}
}
