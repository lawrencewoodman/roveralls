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
	"syscall"
	"testing"
)

func TestRun(t *testing.T) {
	initProgram(os.Args, os.Stdout, os.Stderr, os.Getenv("GOPATH"))
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
		initProgram(c.cmdArgs, &gotOut, &gotErr, os.Getenv("GOPATH"))
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
			t.Errorf("Run: gotErr: %s", gotErr.String())
		}

		if err := checkOutput(c.wantOutRegexps, gotOut.String()); err != nil {
			t.Errorf("checkOutput: %s", err)
		}

		gotFiles, err := filesTested(wd, "roveralls.coverprofile")
		if len(c.wantFiles) != 0 && err != nil {
			t.Fatalf("filesTested err: %s", err)
		}
		if len(gotFiles) != len(c.wantFiles) {
			t.Errorf("Wrong files tested (cmdArgs: %s).  want: %s, got: %v",
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
	initProgram(os.Args, os.Stdout, os.Stderr, os.Getenv("GOPATH"))
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
		initProgram(c.cmdArgs, &gotOut, &gotErr, c.gopath)
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

func TestProcessDir_errors(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	cases := []struct {
		cover   string
		path    string
		wantErr error
	}{
		{cover: "count",
			path:    ".",
			wantErr: errors.New("can't create relative path"),
		},
		{cover: "count",
			path: filepath.Join("fixtures", "nonexistant"),
			wantErr: &os.PathError{
				"chdir",
				filepath.Join("fixtures", "nonexistant"),
				syscall.ENOENT,
			},
		},
		{cover: "bob",
			path: wd,
			wantErr: goTestError{
				stderr: "invalid flag argument for -covermode: \"bob\"",
				stdout: "",
			},
		},
	}
	for i, c := range cases {
		var gotOut bytes.Buffer
		var pOut bytes.Buffer
		program := &Program{cover: c.cover, out: &pOut, verbose: true}
		err := program.processDir(wd, c.path, &gotOut)
		checkErrorMatch(t, fmt.Sprintf("(%d) processDir: ", i), err, c.wantErr)
	}
}

func TestUsage(t *testing.T) {
	var gotErr bytes.Buffer
	initProgram(os.Args, os.Stdout, &gotErr, os.Getenv("GOPATH"))
	Usage()
	want := usageMsg()
	if gotErr.String() != want {
		t.Errorf("Usage: got: %s, want: %s", gotErr.String(), want)
	}
}

func TestGoTestErrorError(t *testing.T) {
	err := goTestError{
		stderr: "this is an error",
		stdout: "baby did a bad bad thing",
	}
	want := "error from go test: this is an error\noutput: baby did a bad bad thing"
	got := err.Error()
	if got != want {
		t.Errorf("Error() got: %s, want: %s", got, want)
	}
}

func TestWalkingErrorError(t *testing.T) {
	err := walkingError{
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

func checkErrorMatch(t *testing.T, context string, got, want error) {
	if got == nil && want == nil {
		return
	}
	if got == nil || want == nil {
		t.Errorf("%s got err: %s, want : %s", context, got, want)
		return
	}
	switch x := want.(type) {
	case *os.PathError:
		if err := checkPathErrorMatch(got, x); err != nil {
			t.Errorf("%s %s", context, err)
		}
		return
	case goTestError:
		if err := checkGoTestErrorMatch(got, x); err != nil {
			t.Errorf("%s %s", context, err)
		}
		return
	}
	if got.Error() != want.Error() {
		t.Errorf("%s got err: %s, want : %s", context, got, want)
	}
}

func checkPathErrorMatch(checkErr error, wantErr *os.PathError) error {
	perr, ok := checkErr.(*os.PathError)
	if !ok {
		return fmt.Errorf("got err type: %T, want error type: os.PathError",
			checkErr)
	}
	if perr.Op != wantErr.Op {
		return fmt.Errorf("got perr.Op: %s, want: %s", perr.Op, wantErr.Op)
	}
	if filepath.Clean(perr.Path) != filepath.Clean(wantErr.Path) {
		return fmt.Errorf("got perr.Path: %s, want: %s", perr.Path, wantErr.Path)
	}
	if perr.Err != wantErr.Err {
		return fmt.Errorf("got perr.Err: %s, want: %s", perr.Err, wantErr.Err)
	}
	return nil
}

func checkGoTestErrorMatch(checkErr error, wantErr goTestError) error {
	gerr, ok := checkErr.(goTestError)
	if !ok {
		return fmt.Errorf("got err type: %T, want error type: goTestError",
			checkErr)
	}
	if strings.Trim(gerr.stderr, " \n") != strings.Trim(wantErr.stderr, " \n") {
		return fmt.Errorf("got gerr.stderr: %s, want: %s",
			gerr.stderr, wantErr.stderr)
	}
	if gerr.stdout != wantErr.stdout {
		return fmt.Errorf("got gerr.stdout: %s, want: %s",
			gerr.stdout, wantErr.stdout)
	}
	return nil
}
