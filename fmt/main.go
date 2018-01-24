package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/hashicorp/hcl2/hclwrite"
	"golang.org/x/crypto/ssh/terminal"
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

	color := terminal.IsTerminal(int(os.Stdout.Fd()))
	width, _, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80
	}
	diagWr := hcl.NewDiagnosticTextWriter(os.Stderr, p.Files(), uint(width), color)

	_, diag := p.ParseHCL(buf, "stdin.hcl")
	if diag != nil && diag.HasErrors() {
		diagWr.WriteDiagnostics(diag)
		return
	}

	out := bytes.TrimSpace(hclwrite.Format(buf))

	fmt.Printf("out:\n---- BEGIN ----\n%s\n---- END ----\n", out)
}
