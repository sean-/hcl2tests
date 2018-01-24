package main

import (
	"fmt"

	"github.com/hashicorp/hcl"
	gohcl2 "github.com/hashicorp/hcl2/gohcl"
	hcl2parse "github.com/hashicorp/hcl2/hclparse"
	"github.com/y0ssar1an/q"
)

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
}
`)

func main() {
	simpleStructTest()
	hclTest()
	hcl2Test()
	hcl2TestSchema()
}

func simpleStructTest() {
	foo := []map[string]interface{}{}
	if err := hcl.Unmarshal(buf, &foo); err != nil {
		panic(fmt.Sprintf("bad: %v", err))
	}

	q.Q(foo)
}

func hclTest() {
	type TestStep struct {
		Name string `hcl:"stepname"`
	}

	type TestCase struct {
		Stepname  string     `hcl:"name",key=casename`
		TestSteps []TestStep `hcl:"step",squash,key=stepname`
	}

	type TestSuite struct {
		Name      string     `hcl:"suitename"`
		TestCases []TestCase `hcl:"testcase"`
	}

	{
		ts := TestSuite{}
		if err := hcl.Unmarshal(buf, &ts); err != nil {
			panic(fmt.Sprintf("bad: %v", err))
		}
		q.Q(ts)
	}

	{
		tc := []TestCase{}
		if err := hcl.Unmarshal(buf, &tc); err != nil {
			panic(fmt.Sprintf("bad: %v", err))
		}
		q.Q(tc)
	}
}

func hcl2Test() {
	type TestStep struct {
		Name string `hcl:"stepname"`
	}

	type TestCase struct {
		Stepname  string     `hcl:"casename"`
		TestSteps []TestStep `hcl:"step,block"`
	}

	type TestSuite struct {
		Name      string     `hcl:"suitename"`
		TestCases []TestCase `hcl:"testcase,block"`
	}

	p := hcl2parse.NewParser()
	f, err := p.ParseHCL(buf, "stdin.hcl")
	if err != nil {
		panic(fmt.Sprintf("bad parse: %v", err))
	}

	ts := TestSuite{}
	diags := gohcl2.DecodeBody(f.Body, nil, &ts)
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
	}

	type TestSuite struct {
		Name      string     `hcl:"suitename"`
		TestCases []TestCase `hcl:"testcase,block"`
	}

	p := hcl2parse.NewParser()
	f, err := p.ParseHCL(buf, "stdin.hcl")
	if err != nil {
		panic(fmt.Sprintf("bad parse: %v", err))
	}

	ts := TestSuite{}
	diags := gohcl2.DecodeBody(f.Body, nil, &ts)
	if diags.HasErrors() {
		panic(fmt.Sprintf("bad decode: %v\n%v", diags, ts))
	}

	q.Q(ts)
}
