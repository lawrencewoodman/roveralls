package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestSubMain(t *testing.T) {
	cases := []struct {
		dir          string
		config       *Config
		wantExitCode int
		wantFiles    []string
	}{
		{dir: "fixtures",
			config: &Config{
				ignore:  ".git,vendor",
				cover:   "count",
				help:    false,
				short:   false,
				verbose: false,
			},
			wantExitCode: 0,
			wantFiles: []string{
				"fixtures/good/good.go",
				"fixtures/good2/good2.go",
			},
		},
		{dir: "fixtures",
			config: &Config{
				ignore:  ".git,vendor",
				cover:   "count",
				help:    false,
				short:   true,
				verbose: false,
			},
			wantExitCode: 0,
			wantFiles: []string{
				"fixtures/good/good.go",
				"fixtures/good2/good2.go",
				"fixtures/short/short.go",
			},
		},
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("ChDir(%s) err: %s", c.dir, err)
		}
		if err := os.Chdir(c.dir); err != nil {
			t.Fatalf("ChDir(%s) err: %s", c.dir, err)
		}
		exitCode := subMain(c.config)
		if exitCode != c.wantExitCode {
			t.Fatalf("subMain() incorrect exit code, got: %d, want: %d",
				exitCode, c.wantExitCode)
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
