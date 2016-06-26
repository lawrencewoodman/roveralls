/*
Package roveralls runs coverage tests on a package and all its sub-packages.  The coverage profile is output as a single file called 'roveralls.coverprofile' for use by tools such as goveralls.

  $ roveralls -help

      usage: roveralls [-covermode mode] [-ignore=dir1,dir2...] [-short] [-v]

      roveralls runs coverage tests on a package and all its sub-packages.  The
      coverage profile is output as a single file called 'roveralls.coverprofile'
      for use by tools such as goveralls.

      The options are:
        -covermode set,count,atomic
        default: count

        -ignore dir1,dir2
        A comma separated list of directory names to ignore, relative to
        working directory.
        default: '.git,vendor'

        -short
        Tell long-running tests to shorten their run time.
        default: false

        -v
        Verbose output.
        default: false
*/
package main
