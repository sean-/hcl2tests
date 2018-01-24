package main

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/y0ssar1an/q"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
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
    stepname = "step2"
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
	Name   string
	Config hcl.Body
}

type TestCase struct {
	Name      string
	Enabled   bool
	TestSteps []*TestStep
	Fixtures  []*TestCaseFixture
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
		Name   string   `hcl:"stepname"`
		Config hcl.Body `hcl:",remain"`
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
				Name:      rtc.Name,
				Fixtures:  []*TestCaseFixture{},
				TestSteps: make([]*TestStep, 0, len(rtc.TestSteps)),
			}

			if rtc.Enabled == nil || *rtc.Enabled == true {
				tc.Enabled = true
			} else {
				tc.Enabled = false
			}

			tc.Fixtures = make([]*TestCaseFixture, 0, len(rtc.Fixtures))
			for _, fixture := range rtc.Fixtures {
				tc.Fixtures = append(tc.Fixtures, &TestCaseFixture{
					Name:   fixture.Name,
					Config: fixture.Config,
				})

				attrs, d := fixture.Config.JustAttributes()
				if d != nil {
					panic(fmt.Sprintf("%+v", d))
				}

				for k, v := range attrs {
					vty, diag := v.Expr.Value(ctx)
					if diag != nil {
						panic(fmt.Sprintf("%+v", diag))
					}
					q.Q("k", k, "v", vty.AsString())

					// The following dump prints:
					//
					// 10) "some_rando"
					// (string) (len=10) "blah BAR 5"
					spew.Dump(k, vty.AsString())
				}
			}

			tc.TestSteps = make([]*TestStep, 0, len(rtc.TestSteps))
			for _, step := range rtc.TestSteps {
				tc.TestSteps = append(tc.TestSteps, &TestStep{
					Name:   step.Name,
					Config: step.Config,
				})

				attrs, d := step.Config.JustAttributes()
				if d != nil {
					panic(fmt.Sprintf("%+v", d))
				}

				for k, v := range attrs {
					vty, diag := v.Expr.Value(ctx)
					if diag != nil {
						panic(fmt.Sprintf("%+v", diag))
					}
					q.Q("k", k, "v", vty.AsString())
					spew.Dump(k, vty.AsString())
				}
			}

			ts.TestCases[i] = tc
		}
	}
}
