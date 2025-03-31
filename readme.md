#### Prerequisites:
- This must be built and ran on a Linux box due to syscall functionality.

#### To build and run:
1. run `go build . -tags linux`
2. to run is similar to the `docker run` command: `./container <command> [bash cmd]`, e.g. `./container run sh`;