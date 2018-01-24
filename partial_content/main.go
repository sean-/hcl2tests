package main

import (
	"fmt"

	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/y0ssar1an/q"
)

func main() {
	hcl2Test()
	hcl2TestSchema()
	hcl2TestSchema2()
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
}

testcase {
  casename = "case2"
  enabled = false
}

`)

func hcl2Test() {
	type TestStep struct {
		Name string `hcl:"stepname"`
	}

	type TestCase struct {
		Stepname  string     `hcl:"casename,attr"`
		TestSteps []TestStep `hcl:"step,block"`
		Enabled   *bool      `hcl:"enabled,attr"`
	}

	type TestSuite struct {
		Name      string     `hcl:"suitename,attr"`
		TestCases []TestCase `hcl:"testcase,block"`
	}

	p := hclparse.NewParser()
	f, err := p.ParseHCL(buf, "stdin.hcl")
	if err != nil {
		panic(fmt.Sprintf("bad parse: %v", err))
	}

	ts := TestSuite{}
	diags := gohcl.DecodeBody(f.Body, nil, &ts)
	if diags.HasErrors() {
		panic(fmt.Sprintf("bad decode: %v\n%v", diags, ts))
	}

	q.Q(ts)
}

func hcl2TestSchema() {
	type TestStep struct {
		Name string `hcl:"stepname"`
	}

	type TestCase struct {
		Stepname  string     `hcl:"casename"`
		TestSteps []TestStep `hcl:"step,block"`
		Enabled   *bool      `hcl:"enabled"`
	}

	type TestSuite struct {
		Name      string     `hcl:"suitename"`
		TestCases []TestCase `hcl:"testcase,block"`
	}

	p := hclparse.NewParser()
	f, err := p.ParseHCL(buf, "stdin.hcl")
	if err != nil {
		panic(fmt.Sprintf("bad parse: %v", err))
	}

	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{
				Name:     "suitename",
				Required: true,
			},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{
				Type:       "testcase",
				LabelNames: []string{},
			},
		},
	}

	content, remain, diags := f.Body.PartialContent(schema)
	if diags.HasErrors() {
		panic(fmt.Sprintf("bad schema: %v\n%v\n%v", content, remain, diags))
	}
	q.Q(content, remain, diags)

	ts := TestSuite{}
	diags = gohcl.DecodeBody(f.Body, nil, &ts)
	if diags.HasErrors() {
		panic(fmt.Sprintf("bad decode: %v\n%v", diags, ts))
	}

	q.Q(ts)
}

func hcl2TestSchema2() {
	type TestStep struct {
		Name string `hcl:"stepname,attr"`
	}

	type TestCase struct {
		Stepname  string     `hcl:"casename"`
		TestSteps []TestStep `hcl:"step,block"`
		Enabled   *bool      `hcl:"enabled,attr"`
	}

	type TestSuite struct {
		Name      string     `hcl:"suitename"`
		TestCases []TestCase `hcl:"testcase,block"`
	}

	schema, partial := gohcl.ImpliedBodySchema(&TestSuite{})
	q.Q(schema, partial)

	p := hclparse.NewParser()
	f, err := p.ParseHCL(buf, "stdin.hcl")
	if err != nil {
		panic(fmt.Sprintf("bad parse: %v", err))
	}

	content, remain, diags := f.Body.PartialContent(schema)
	if diags.HasErrors() {
		panic(fmt.Sprintf("bad schema: %v\n%v\n%v", content, remain, diags))
	}
	q.Q(content, remain, diags)

	ts := TestSuite{}
	diags = gohcl.DecodeBody(f.Body, nil, &ts)
	if diags.HasErrors() {
		panic(fmt.Sprintf("bad decode: %v\n%v", diags, ts))
	}

	q.Q(ts)
}
