/*
Package curator is a tool for generating and publishing operating
system packages (e.g. deb and rpm) to S3 buckets.

# Architecture and Organization

The curator binary is built from the "main/curator.go" package, with a
command that resembles the following:

	go build -o curator main/curator.go

See the "makefile" in the curator repository for additional build and
test automation. The command line interface uses the urfave/cli
package, with the implementation of entry points in the "operations"
package.
*/
package curator
