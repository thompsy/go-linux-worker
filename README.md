# Go Linux Worker

[![Build Status](https://img.shields.io/github/workflow/status/thompsy/go-linux-worker/build/master)](https://github.com/thompsy/go-linux-worker/actions/workflows/build.yaml?query=branch%3Amaster)
[![Go Report Card](https://goreportcard.com/badge/github.com/thompsy/go-linux-workerg)](https://goreportcard.com/report/github.com/thompsy/go-linux-worker)

The Go Linux Worker Service executes arbitrary Linux commands on behalf of clients. Processes are isolated to a basic Apline Linux container using Linux `namespaces` and resource constraints are provided using `cgroups`. 

The primary purpose of this project was to allow me to experiment with `namespaces` and `cgroups` to better understand how containers work under the hood.

The design of the service can be seen in the [design document](DESIGN.md).

Currently, the service consists of a library with functions to submit a job, query the status of a job, stop a job and fetch the logs from a job along with client and server implementations.

The code has been developed and tested on Go v1.15.

## Building

A `Makefile` is provided to build, test and check the code. In order to check the code the [golangci-lint](golangci-lint.run) tool is required. This can be installed into the `./bin/` directory by running the following command:

    make install-tools

Once the required tools are installed the following command can be used to build, test and lint the project:

    make all

Alternatively, a Docker image can be built and run using the following command:

    make docker-build

## Running
The server can be run using the built Docker image by running:

    make docker-run

The client can be run using:

    # Submit a job
    $ ./bin/client submit "ls -lah /"
    Id of submitted job: 58403eaf-6691-4796-9b9c-cb60d9dc1763

    # Get the status of the job
    $ ./bin/client status 58403eaf-6691-4796-9b9c-cb60d9dc1763
    Status: COMPLETED
    Exit code: 0

    # Get the output from the job
    $ ./bin/client logs 58403eaf-6691-4796-9b9c-cb60d9dc1763
    total 0
    drwxr-xr-x   19 root     root         380 Apr 26 19:48 .
    drwxr-xr-x   19 root     root         380 Apr 26 19:48 ..
    drwxr-xr-x    2 root     root        1.6K Apr 26 19:48 bin
    drwxr-xr-x    2 root     root          40 Feb 17 15:07 dev
    drwxr-xr-x   15 root     root         700 Apr 26 19:48 etc
    drwxr-xr-x    2 root     root          40 Feb 17 15:07 home
    drwxr-xr-x    7 root     root         280 Apr 26 19:48 lib
    drwxr-xr-x    5 root     root         100 Apr 26 19:48 media
    drwxr-xr-x    2 root     root          40 Feb 17 15:07 mnt
    drwxr-xr-x    2 root     root          40 Feb 17 15:07 opt
    dr-xr-xr-x  171 root     root           0 Apr 26 19:48 proc
    drwx------    2 root     root          40 Feb 17 15:07 root
    drwxr-xr-x    2 root     root          40 Feb 17 15:07 run
    drwxr-xr-x    2 root     root        1.2K Apr 26 19:48 sbin
    drwxr-xr-x    2 root     root          40 Feb 17 15:07 srv
    drwxr-xr-x    2 root     root          40 Feb 17 15:07 sys
    drwxrwxrwt    2 root     root          40 Feb 17 15:07 tmp
    drwxr-xr-x    7 root     root         140 Apr 26 19:48 usr
    drwxr-xr-x   12 root     root         260 Apr 26 19:48 var

    # Submit a job
    $ ./bin/client sugmit "sleep 10m"
    Id of submitted job: fc5aa11b-7f1c-4435-8c7b-aee3e0f09b7f

    # Get the status of the job
    $ ./bin/client status fc5aa11b-7f1c-4435-8c7b-aee3e0f09b7f
    Status: RUNNING

    # Stop the job
    $ ./bin/client stop fc5aa11b-7f1c-4435-8c7b-aee3e0f09b7f
    INFO[0000] fc5aa11b-7f1c-4435-8c7b-aee3e0f09b7f: killed

    # Get the status of the job
    $ ./bin/client status fc5aa11b-7f1c-4435-8c7b-aee3e0f09b7f
    Status: STOPPED
