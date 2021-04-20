# Worker-API Service

The Worker-API Service executes arbitrary Linux commands on behalf of clients. The design of the service can be seen in the [design document](DESIGN.md).

Currently, the service consists of a library with functions to submit a job, query the status of a job, stop a job and fetch the logs from a job. Future releases will provide a `gRPC` client and server built on top of this library.

The code has been developed and tested on Go v1.15.

## Building

A `Makefile` is provided to build, test and check the code. In order to check the code the [golangci-lint](golangci-lint.run) tool is required. This can be installed into the `./bin/` directory by running the following command:

    make install-tools

Once the required tools are installed the following command can be used to build, test and lint the project:

    make all

