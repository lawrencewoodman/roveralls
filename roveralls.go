// Copyright (c) 2016 Lawrence Woodman <lwoodman@vlifesystems.com>
// Licensed under an MIT licence.  Please see LICENCE.md for details.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// This is a horrible kludge so that errors can be tested properly
var program *Program

// Usage is used by flag package if an error occurs when parsing flags
var Usage = func() {
	subUsage(program.outErr)
}

func subUsage(out io.Writer) {
	fmt.Fprintf(out, usageMsg())
}

func usageMsg() string {
	var b bytes.Buffer
	const desc = `
roveralls runs coverage tests on a package and all its sub-packages.  The
coverage profile is output as a single file called 'roveralls.coverprofile'
for use by tools such as goveralls.
`
	fmt.Fprintf(&b, "%s\n", desc)
	fmt.Fprintf(&b, "Usage:\n")
	program.flagSet.SetOutput(&b)
	program.flagSet.PrintDefaults()
	program.flagSet.SetOutput(program.outErr)
	return b.String()
}

func usagePartialMsg() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "Usage:\n")
	program.flagSet.SetOutput(&b)
	program.flagSet.PrintDefaults()
	program.flagSet.SetOutput(program.outErr)
	return b.String()
}

const (
	defaultIgnores = ".git,vendor"
	outFilename    = "roveralls.coverprofile"
)

type goTestError struct {
	err    error
	output string
}

func (e goTestError) Error() string {
	return fmt.Sprintf("error from go test: %s\noutput: %s",
		e.err, e.output)
}

type WalkingError struct {
	dir string
	err error
}

func (e WalkingError) Error() string {
	return fmt.Sprintf("could not walk working directory '%s': %s",
		e.dir, e.err)
}

type Program struct {
	ignore  string
	cover   string
	help    bool
	short   bool
	verbose bool
	ignores map[string]bool
	cmdArgs []string
	flagSet *flag.FlagSet
	out     io.Writer
	outErr  io.Writer
	gopath  string
}

func InitProgram(
	cmdArgs []string,
	out io.Writer,
	outErr io.Writer,
	gopath string,
) {
	program = &Program{out: out, outErr: outErr, cmdArgs: cmdArgs, gopath: gopath}
	program.initFlagSet()
}

func (p *Program) Run() int {
	if err := p.flagSet.Parse(p.cmdArgs[1:]); err != nil {
		return 1
	}
	if isProblem := p.handleGOPATH(); isProblem {
		return 1
	}

	if isProblem := p.handleFlags(); isProblem {
		return 1
	}
	if p.help {
		subUsage(p.out)
		return 0
	}

	if err := p.testCoverage(); err != nil {
		fmt.Fprintf(p.outErr, "\n%s\n", err)
		return 1
	}
	return 0
}

func (p *Program) ignoreDir(relDir string) bool {
	_, ignore := p.ignores[relDir]
	return ignore
}

func (p *Program) initFlagSet() {
	p.flagSet = flag.NewFlagSet("", flag.ContinueOnError)
	p.flagSet.SetOutput(p.outErr)
	p.flagSet.StringVar(
		&p.cover,
		"covermode",
		"count",
		"Mode to run when testing files: `count,set,atomic`",
	)
	p.flagSet.StringVar(
		&p.ignore,
		"ignore",
		defaultIgnores,
		"Comma separated list of directory names to ignore: `dir1,dir2,...`",
	)
	p.flagSet.BoolVar(&p.verbose, "v", false, "Verbose output")
	p.flagSet.BoolVar(
		&p.short,
		"short",
		false,
		"Tell long-running tests to shorten their run time",
	)
	p.flagSet.BoolVar(&p.help, "help", false, "Display this help")
}

// returns true if a problem, else false
func (p *Program) handleGOPATH() bool {
	gopath := filepath.Clean(p.gopath)
	if p.verbose {
		fmt.Fprintln(p.out, "GOPATH:", gopath)
	}

	if len(gopath) == 0 || gopath == "." {
		fmt.Fprintf(p.outErr, "invalid GOPATH '%s'\n", gopath)
		return true
	}
	return false
}

// returns true if a problem, else false
func (p *Program) handleFlags() bool {
	validCoverModes := map[string]bool{"set": true, "count": true, "atomic": true}
	if _, ok := validCoverModes[p.cover]; !ok {
		fmt.Fprintf(p.outErr, "invalid covermode '%s'\n", p.cover)
		subUsage(p.outErr)
		return true
	}

	arr := strings.Split(p.ignore, ",")
	p.ignores = make(map[string]bool, len(arr))
	for _, v := range arr {
		p.ignores[v] = true
	}
	return false
}

var modeRegexp = regexp.MustCompile("mode: [a-z]+\n")

func (p *Program) testCoverage() error {
	var buff bytes.Buffer

	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	if p.verbose {
		fmt.Fprintln(p.out, "Working dir:", wd)
	}

	walker := p.makeWalker(wd, &buff)
	if err := filepath.Walk(wd, walker); err != nil {
		return WalkingError{
			dir: wd,
			err: err,
		}
	}

	final := buff.String()
	final = modeRegexp.ReplaceAllString(final, "")
	final = fmt.Sprintf("mode: %s\n%s", p.cover, final)

	if err := ioutil.WriteFile(outFilename, []byte(final), 0644); err != nil {
		return fmt.Errorf("error writing to: %s, %s", outFilename, err)
	}
	return nil
}

func (p *Program) makeWalker(
	wd string,
	buff *bytes.Buffer,
) func(string, os.FileInfo, error) error {
	return func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(wd, path)
		if err != nil {
			return fmt.Errorf("error creating relative path")
		}

		if p.ignoreDir(rel) {
			return filepath.SkipDir
		}

		files, err := filepath.Glob(filepath.Join(path, "*_test.go"))
		if err != nil {
			return fmt.Errorf("error checking for test files")
		}
		if len(files) == 0 {
			if p.verbose {
				fmt.Fprintf(p.out, "No Go test files in dir: %s, skipping\n", rel)
			}
			return nil
		}
		return p.processDir(wd, path, buff)
	}
}

func (p *Program) processDir(wd string, path string, buff *bytes.Buffer) error {
	var cmd *exec.Cmd
	var cmdOut bytes.Buffer
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := os.Chdir(path); err != nil {
		return err
	}
	defer os.Chdir(wd)

	outDir, err := ioutil.TempDir("", "roveralls")
	if err != nil {
		return err
	}
	defer os.RemoveAll(outDir)

	if p.verbose {
		rel, err := filepath.Rel(wd, path)
		if err != nil {
			return fmt.Errorf("error creating relative path")
		}
		fmt.Fprintf(p.out, "Processing dir: %s\n", rel)
		if p.short {
			fmt.Fprintf(p.out,
				"Processing: go test -short -covermode=%s -coverprofile=profile.coverprofile -outputdir=%s\n",
				p.cover, outDir)
		} else {
			fmt.Fprintf(p.out,
				"Processing: go test -covermode=%s -coverprofile=profile.coverprofile -outputdir=%s\n",
				p.cover, outDir)
		}
	}

	if p.short {
		cmd = exec.Command("go",
			"test",
			"-short",
			"-covermode="+p.cover,
			"-coverprofile=profile.coverprofile",
			"-outputdir="+outDir,
		)
	} else {
		cmd = exec.Command("go",
			"test",
			"-covermode="+p.cover,
			"-coverprofile=profile.coverprofile",
			"-outputdir="+outDir,
		)
	}
	cmd.Stdout = &cmdOut
	if err := cmd.Run(); err != nil {
		return goTestError{
			err:    err,
			output: cmdOut.String(),
		}
	}

	b, err := ioutil.ReadFile(filepath.Join(outDir, "profile.coverprofile"))
	if err != nil {
		return err
	}

	_, err = buff.Write(b)
	return err
}

func main() {
	InitProgram(os.Args, os.Stdout, os.Stderr, os.Getenv("GOPATH"))
	os.Exit(program.Run())
}
