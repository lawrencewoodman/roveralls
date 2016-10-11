// Copyright (c) 2016 Lawrence Woodman <lwoodman@vlifesystems.com>
// Licensed under an MIT licence.  Please see LICENCE.md for details.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var Usage = func() {
	const desc = `
roveralls runs coverage tests on a package and all its sub-packages.  The
coverage profile is output as a single file called 'roveralls.coverprofile'
for use by tools such as goveralls.
`
	fmt.Fprintf(os.Stderr, "%s\n", desc)
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

const (
	defaultIgnores = ".git,vendor"
	outFilename    = "roveralls.coverprofile"
)

type ErrInvalidCoverMode string

func (e ErrInvalidCoverMode) Error() string {
	return fmt.Sprintf("invalid covermode '%s'", string(e))
}

type ErrGoTest struct {
	err    error
	output string
}

func (e ErrGoTest) Error() string {
	return fmt.Sprintf("error from go test: %s\noutput: %s",
		e.err, e.output)
}

type ErrWalking struct {
	dir string
	err error
}

func (e ErrWalking) Error() string {
	return fmt.Sprintf("could not walk working directory '%s'\n%s\n",
		e.dir, e.err)
}

type Config struct {
	ignore  string
	cover   string
	help    bool
	short   bool
	verbose bool
	ignores map[string]bool
}

func (c *Config) ignoreDir(relDir string) bool {
	_, ignore := c.ignores[relDir]
	return ignore
}

func main() {
	config := parseFlags()
	os.Exit(subMain(config))
}

func subMain(config *Config) int {
	l := log.New(os.Stderr, "", 0)

	if err := handleFlags(config); err != nil {
		l.Printf("\n%s\n", err)
		if _, ok := err.(ErrInvalidCoverMode); ok {
			Usage()
		}
		return 1
	}
	if config.help {
		return 0
	}

	if err := testCoverage(config); err != nil {
		l.Printf("\n%s\n", err)
		return 1
	}
	return 0
}

func parseFlags() *Config {
	config := &Config{}
	flag.StringVar(
		&config.cover,
		"covermode",
		"count",
		"Mode to run when testing files: `count,set,atomic`",
	)
	flag.StringVar(
		&config.ignore,
		"ignore",
		defaultIgnores,
		"Comma separated list of directory names to ignore: `dir1,dir2,...`",
	)
	flag.BoolVar(&config.verbose, "v", false, "Verbose output")
	flag.BoolVar(
		&config.short,
		"short",
		false,
		"Tell long-running tests to shorten their run time",
	)
	flag.BoolVar(&config.help, "help", false, "Display this help")
	flag.Parse()
	return config
}

func handleFlags(config *Config) error {
	gopath := filepath.Clean(os.Getenv("GOPATH"))
	if config.help {
		Usage()
		return nil
	}

	if config.verbose {
		fmt.Println("GOPATH:", gopath)
	}

	if len(gopath) == 0 || gopath == "." {
		return fmt.Errorf("invalid GOPATH '%s'", gopath)
	}

	validCoverModes := map[string]bool{"set": true, "count": true, "atomic": true}
	if _, ok := validCoverModes[config.cover]; !ok {
		return ErrInvalidCoverMode(config.cover)
	}

	arr := strings.Split(config.ignore, ",")
	config.ignores = make(map[string]bool, len(arr))
	for _, v := range arr {
		config.ignores[v] = true
	}
	return nil
}

var modeRegexp = regexp.MustCompile("mode: [a-z]+\n")

func testCoverage(config *Config) error {
	var buff bytes.Buffer

	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	if config.verbose {
		fmt.Println("Working dir:", wd)
	}

	walker := makeWalker(wd, &buff, config)
	if err := filepath.Walk(wd, walker); err != nil {
		return ErrWalking{
			dir: wd,
			err: err,
		}
	}

	final := buff.String()
	final = modeRegexp.ReplaceAllString(final, "")
	final = fmt.Sprintf("mode: %s\n%s", config.cover, final)

	if err := ioutil.WriteFile(outFilename, []byte(final), 0644); err != nil {
		return fmt.Errorf("error writing to: %s, %s", outFilename, err)
	}
	return nil
}

func makeWalker(
	wd string,
	buff *bytes.Buffer,
	config *Config,
) func(string, os.FileInfo, error) error {
	return func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(wd, path)
		if err != nil {
			return fmt.Errorf("error creating relative path")
		}

		if config.ignoreDir(rel) {
			return filepath.SkipDir
		}

		files, err := filepath.Glob(filepath.Join(path, "*_test.go"))
		if err != nil {
			return fmt.Errorf("error checking for test files")
		}
		if len(files) == 0 {
			if config.verbose {
				fmt.Printf("No Go Test files in dir: %s, skipping\n", rel)
			}
			return nil
		}
		return processDir(path, config, buff)
	}
}

func processDir(path string, config *Config, buff *bytes.Buffer) error {
	var cmdOut bytes.Buffer
	shortFlagStr := ""
	if config.short {
		shortFlagStr = "-short"
	}
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

	if config.verbose {
		fmt.Printf("Processing: go test %s -covermode=%s -coverprofile=profile.coverprofile -outputdir=%s\n",
			shortFlagStr, config.cover, outDir)
	}

	cmd := exec.Command("go",
		"test",
		shortFlagStr,
		"-covermode="+config.cover,
		"-coverprofile=profile.coverprofile",
		"-outputdir="+outDir,
	)
	cmd.Stdout = &cmdOut
	if err := cmd.Run(); err != nil {
		return ErrGoTest{
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
