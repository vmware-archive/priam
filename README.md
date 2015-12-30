Start of command line client for VMware Workspace in Go.

Build it:

    go build

Test it (basic or with code coverage):

    go test
    
    go test -cover

Test it and view code coverage in a browser:

    go test -coverprofile=coverage.out
    go tool cover -html=coverage.out

To get even fancier with heat maps of code coverage, see http://blog.golang.org/cover

Install it:

    go install


get help:

    priam help

get help on specific command (for example, 'target'):

    priam help target
