/*
Package roveralls runs coverage tests on a package and all its sub-packages.  The coverage profile is output as a single file called 'roveralls.coverprofile' for use by tools such as goveralls.

  $ roveralls -help

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
*/
package main
