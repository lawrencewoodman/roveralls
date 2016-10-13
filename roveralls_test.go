package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestSubMain(t *testing.T) {
	cases := []struct {
		dir          string
		cmdArgs      []string
		wantExitCode int
		wantFiles    []string
	}{
		{dir: "fixtures",
			cmdArgs:      []string{os.Args[0], "-covermode=count"},
			wantExitCode: 0,
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
			wantExitCode: 0,
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
			wantExitCode: 0,
			wantFiles: []string{
				filepath.Join("fixtures", "good", "good.go"),
				filepath.Join("fixtures", "good2", "good2.go"),
				filepath.Join("fixtures", "short", "short.go"),
			},
		},
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	for _, c := range cases {
		var gotErr bytes.Buffer
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("ChDir(%s) err: %s", c.dir, err)
		}
		if err := os.Chdir(c.dir); err != nil {
			t.Fatalf("ChDir(%s) err: %s", c.dir, err)
		}
		exitCode := subMain(c.cmdArgs, &gotErr)
		if exitCode != c.wantExitCode {
			t.Fatalf("subMain: incorrect exit code, got: %d, want: %d",
				exitCode, c.wantExitCode)
		}

		if gotErr.String() != "" {
			t.Fatalf("subMain: gotErr: %s", gotErr)
		}

		gotFiles, err := filesTested(wd, "roveralls.coverprofile")
		if err != nil {
			t.Fatalf("filesTested err: %s", err)
		}
		if len(gotFiles) != len(c.wantFiles) {
			t.Fatalf("Wrong files tested.  want: %s, got: %s", c.wantFiles, gotFiles)
		}
		for _, wantFile := range c.wantFiles {
			if _, ok := gotFiles[wantFile]; !ok {
				t.Fatalf("No cover entries for file: %s", wantFile)
			}
		}
	}
}

func TestSubMain_errors(t *testing.T) {
	cases := []struct {
		dir          string
		cmdArgs      []string
		wantExitCode int
		wantErr      string
	}{
		{dir: "fixtures",
			cmdArgs:      []string{os.Args[0], "-covermode=nothing"},
			wantExitCode: 1,
			wantErr:      "invalid covermode 'nothing'\n" + usageMsg(),
		},
		{dir: "fixtures",
			cmdArgs:      []string{os.Args[0], "-bob"},
			wantExitCode: 2,
			wantErr:      "flag provided but not defined: -bob\n" + usagePartialMsg(),
		},
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	for _, c := range cases {
		var gotErr bytes.Buffer
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("ChDir(%s) err: %s", c.dir, err)
		}
		if err := os.Chdir(c.dir); err != nil {
			t.Fatalf("ChDir(%s) err: %s", c.dir, err)
		}
		exitCode := subMain(c.cmdArgs, &gotErr)
		if exitCode != c.wantExitCode {
			t.Errorf("subMain: incorrect exit code, got: %d, want: %d",
				exitCode, c.wantExitCode)
		}

		if gotErr.String() != c.wantErr {
			t.Errorf("subMain: gotErr: %s, wantErr: %s", gotErr.String(), c.wantErr)
		}
	}
}

func TestInvalidCoverModeErrorError(t *testing.T) {
	err := InvalidCoverModeError("fred")
	want := "invalid covermode 'fred'"
	got := err.Error()
	if got != want {
		t.Errorf("Error() got: %s, want: %s", got, want)
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
