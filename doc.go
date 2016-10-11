/*
A recursive coverage testing tool.

roveralls runs coverage tests on a package and all its sub-packages.  The coverage profile is output as a single file called 'roveralls.coverprofile' for use by tools such as goveralls.

This tool was inspired by https://github.com/go-playground/overalls written by Dean Karn, but I found it difficult to test and brittle so I decided to rewrite it from scratch.  Thanks for the inspiration Dean.

Usage

At its simplest, to test the current package and sub-packages and create a 'roveralls.coverprofile' file in the directory that you run the command:

    roveralls

To see the help for the command:

    roveralls -help

      roveralls runs coverage tests on a package and all its sub-packages.  The
      coverage profile is output as a single file called 'roveralls.coverprofile'
      for use by tools such as goveralls.

      Usage of roveralls:
        -covermode count,set,atomic
            Mode to run when testing files: count,set,atomic (default "count")
        -help
            Display this help
        -ignore dir1,dir2,...
            Comma separated list of directory names to ignore: dir1,dir2,... (default ".git,vendor")
        -short
            Tell long-running tests to shorten their run time
        -v	Verbose output

View Output in a Web Browser

To view the code coverage for you packge in a browser:

    go tool cover -html=roveralls.coverprofile

Use with goveralls

The output of roveralls is the same as the the standard:
  go test -coverprofile=profile.coverprofile
but with multiple files tested in the output file.  This can therefore be used with tools such as goveralls.

If you wanted to call it from a '.travis.yml' script you could use:

    - $HOME/gopath/bin/roveralls
    - $HOME/gopath/bin/goveralls -coverprofile=roveralls.coverprofile -service=travis-ci
*/
package main
