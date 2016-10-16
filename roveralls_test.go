package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	InitProgram(os.Args, os.Stdout, os.Stderr, os.Getenv("GOPATH"))
	cases := []struct {
		dir            string
		cmdArgs        []string
		wantExitCode   int
		wantOutRegexps []string
		wantFiles      []string
	}{
		{dir: "fixtures",
			cmdArgs:        []string{os.Args[0], "-covermode=count"},
			wantExitCode:   0,
			wantOutRegexps: []string{},
			wantFiles: []string{
				filepath.Join("fixtures", "good", "good.go"),
				filepath.Join("fixtures", "good2", "good2.go"),
			},
		},
		{dir: "fixtures",
			cmdArgs:      []string{os.Args[0], "-covermode=count", "-v"},
			wantExitCode: 0,
			wantOutRegexps: []string{
				"^GOPATH: .*$",
				"^Working dir: .*$",
				"^No Go test files in dir: ., skipping$",
				"^Processing dir: good$",
				"^Processing: go test -covermode=count -coverprofile=profile.coverprofile -outputdir=.*$",
				"^Processing dir: good2$",
				"^Processing: go test -covermode=count -coverprofile=profile.coverprofile -outputdir=.*$",
				"^No Go test files in dir: no-go-files, skipping$",
				"^No Go test files in dir: no-test-files, skipping$",
				"^Processing dir: short$",
				"^Processing: go test -covermode=count -coverprofile=profile.coverprofile -outputdir=.*$",
			},
			wantFiles: []string{
				filepath.Join("fixtures", "good", "good.go"),
				filepath.Join("fixtures", "good2", "good2.go"),
			},
		},
		{dir: "fixtures",
			cmdArgs: []string{
				os.Args[0],
				"-covermode=count",
				"-ignore=.git,vendor,good2",
				"-short",
			},
			wantExitCode:   0,
			wantOutRegexps: []string{},
			wantFiles: []string{
				filepath.Join("fixtures", "good", "good.go"),
				filepath.Join("fixtures", "short", "short.go"),
			},
		},
		{dir: "fixtures",
			cmdArgs: []string{
				os.Args[0],
				"-covermode=count",
				"-ignore=.git,vendor,good2",
				"-v",
				"-short",
			},
			wantExitCode: 0,
			wantOutRegexps: []string{
				"^GOPATH: .*$",
				"^Working dir: .*$",
				"^No Go test files in dir: ., skipping$",
				"^Processing dir: good$",
				"^Processing: go test -short -covermode=count -coverprofile=profile.coverprofile -outputdir=.*$",
				"^No Go test files in dir: no-go-files, skipping$",
				"^No Go test files in dir: no-test-files, skipping$",
				"^Processing dir: short$",
				"^Processing: go test -short -covermode=count -coverprofile=profile.coverprofile -outputdir=.*$",
			},
			wantFiles: []string{
				filepath.Join("fixtures", "good", "good.go"),
				filepath.Join("fixtures", "short", "short.go"),
			},
		},
		{dir: "fixtures",
			cmdArgs: []string{
				os.Args[0],
				"-covermode=count",
				"-short",
			},
			wantExitCode:   0,
			wantOutRegexps: []string{},
			wantFiles: []string{
				filepath.Join("fixtures", "good", "good.go"),
				filepath.Join("fixtures", "good2", "good2.go"),
				filepath.Join("fixtures", "short", "short.go"),
			},
		},
		{dir: "fixtures",
			cmdArgs:        []string{os.Args[0], "-help"},
			wantExitCode:   0,
			wantOutRegexps: makeUsageMsgRegexps(),
			wantFiles:      []string{},
		},
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	for _, c := range cases {
		var gotOut bytes.Buffer
		var gotErr bytes.Buffer
		InitProgram(c.cmdArgs, &gotOut, &gotErr, os.Getenv("GOPATH"))
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("ChDir(%s) err: %s", c.dir, err)
		}
		if err := os.Chdir(c.dir); err != nil {
			t.Fatalf("ChDir(%s) err: %s", c.dir, err)
		}
		os.Remove(filepath.Join("roveralls.coverprofile"))
		exitCode := program.Run()
		if exitCode != c.wantExitCode {
			t.Errorf("Run: incorrect exit code, got: %d, want: %d",
				exitCode, c.wantExitCode)
		}

		if gotErr.String() != "" {
			t.Errorf("Run: gotErr: %s", gotErr)
		}

		if err := checkOutput(c.wantOutRegexps, gotOut.String()); err != nil {
			t.Errorf("checkOutput: %s", err)
		}

		gotFiles, err := filesTested(wd, "roveralls.coverprofile")
		if len(c.wantFiles) != 0 && err != nil {
			t.Fatalf("filesTested err: %s", err)
		}
		if len(gotFiles) != len(c.wantFiles) {
			t.Errorf("Wrong files tested (cmdArgs: %s).  want: %s, got: %s",
				c.cmdArgs, c.wantFiles, gotFiles)
		}
		for _, wantFile := range c.wantFiles {
			if _, ok := gotFiles[wantFile]; !ok {
				t.Errorf("No cover entries for file: %s", wantFile)
			}
		}
	}
}

func TestRun_errors(t *testing.T) {
	InitProgram(os.Args, os.Stdout, os.Stderr, os.Getenv("GOPATH"))
	cases := []struct {
		dir          string
		cmdArgs      []string
		gopath       string
		wantExitCode int
		wantOut      string
		wantErr      string
	}{
		{dir: "fixtures",
			cmdArgs:      []string{os.Args[0], "-covermode=nothing"},
			gopath:       os.Getenv("GOPATH"),
			wantExitCode: 1,
			wantOut:      "",
			wantErr:      "invalid covermode 'nothing'\n" + usageMsg(),
		},
		{dir: "fixtures",
			cmdArgs:      []string{os.Args[0], "-bob"},
			gopath:       os.Getenv("GOPATH"),
			wantExitCode: 1,
			wantOut:      "",
			wantErr:      "flag provided but not defined: -bob\n" + usagePartialMsg(),
		},
		{dir: "fixtures",
			cmdArgs:      []string{os.Args[0], "-covermode=count"},
			gopath:       "",
			wantExitCode: 1,
			wantOut:      "",
			wantErr:      "invalid GOPATH '.'\n",
		},
		{dir: "fixtures",
			cmdArgs:      []string{os.Args[0], "-covermode=count"},
			gopath:       ".",
			wantExitCode: 1,
			wantOut:      "",
			wantErr:      "invalid GOPATH '.'\n",
		},
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	for _, c := range cases {
		var gotOut bytes.Buffer
		var gotErr bytes.Buffer
		InitProgram(c.cmdArgs, &gotOut, &gotErr, c.gopath)
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("ChDir(%s) err: %s", c.dir, err)
		}
		if err := os.Chdir(c.dir); err != nil {
			t.Fatalf("ChDir(%s) err: %s", c.dir, err)
		}
		exitCode := program.Run()
		if exitCode != c.wantExitCode {
			t.Errorf("Run: incorrect exit code, got: %d, want: %d",
				exitCode, c.wantExitCode)
		}

		if gotErr.String() != c.wantErr {
			t.Errorf("Run: gotErr: %s, wantErr: %s", gotErr.String(), c.wantErr)
		}

		if gotOut.String() != c.wantOut {
			t.Errorf("Run: gotOut: %s, wantOut: %s", gotOut.String(), c.wantOut)
		}
	}
}

func TestGoTestErrorError(t *testing.T) {
	err := GoTestError{
		err:    errors.New("this is an error"),
		output: "baby did a bad bad thing",
	}
	want := "error from go test: this is an error\noutput: baby did a bad bad thing"
	got := err.Error()
	if got != want {
		t.Errorf("Error() got: %s, want: %s", got, want)
	}
}

func TestWalkingErrorError(t *testing.T) {
	err := WalkingError{
		err: errors.New("this is an error"),
		dir: "/tmp/someplace",
	}
	want := "could not walk working directory '/tmp/someplace': this is an error"
	got := err.Error()
	if got != want {
		t.Errorf("Error() got: %s, want: %s", got, want)
	}
}

/****************************
 *  Helper functions
 ****************************/

var fileTestedRegexp = regexp.MustCompile("^(.*?)(:\\d.*) (\\d+)$")

func makeUsageMsgRegexps() []string {
	lines := strings.Split(usageMsg(), "\n")[:15]
	r := make([]string, len(lines))
	for i, l := range lines {
		r[i] = regexp.QuoteMeta(l)
	}
	return r
}

func checkOutput(wantRegexp []string, gotOut string) error {
	gotOutStrs := strings.Split(gotOut, "\n")
	gotOutStrs = gotOutStrs[:len(gotOutStrs)-1]
	if len(wantRegexp) != len(gotOutStrs) {
		return fmt.Errorf("wantRegexp has %d lines, gotOut has %d lines",
			len(wantRegexp), len(gotOutStrs))
	}
	i := 0
	for _, r := range wantRegexp {
		compiledRegexp, err := regexp.Compile(r)
		if err != nil {
			return err
		}
		if !compiledRegexp.MatchString(gotOutStrs[i]) {
			return fmt.Errorf("line doesn't match got: %s, want: %s",
				gotOutStrs[i], r)
		}
		i++
	}
	return nil
}

func filesTested(wd string, filename string) (map[string]bool, error) {
	files := map[string]bool{}
	gopath := filepath.Clean(os.Getenv("GOPATH"))
	if len(gopath) == 0 {
		return files, fmt.Errorf("invalid GOPATH '%s'", gopath)
	}
	srcpath := filepath.Join(gopath, "src")
	project, err := filepath.Rel(srcpath, wd)
	if err != nil {
		return files, err
	}
	project = filepath.Clean(project)
	f, err := os.Open(filename)
	if err != nil {
		return files, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if fileTestedRegexp.MatchString(line) {
			file := fileTestedRegexp.ReplaceAllString(line, "$1")
			file, err = filepath.Rel(project, file)
			if err != nil {
				return files, err
			}
			count := fileTestedRegexp.ReplaceAllString(line, "$3")
			if count != "0" {
				files[file] = true
			}
		}
	}
	return files, scanner.Err()
}
