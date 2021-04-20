# Worker API Service

This document describes the design of the Worker-API service which executes arbitrary Linux commands on behalf of clients. The service consists of the following components: a library, a server and a client, each of which is described below.

## Server
The service provided by the server is defined below:

    service WorkerService {
    	rpc Submit (Command) returns (JobId) {}
    	rpc Stop (JobId) returns (Empty) {}
    	rpc Status (JobId) returns (StatusResponse) {}
    	rpc GetLogs (JobId) returns (stream Log) {}
    }

The `Submit` call takes a `Command` and returns a `JobID`.

    message Command {
    	string command = 1;
    }

A command is simply a string which should include the command itself along with any arguments. A client may submit a single command at a time. Depending on the type of workloads expected it could be more efficient to allow clients to submit multiple commands at a time however that is beyond the scope of this implementation.

    message JobId {
    	string id = 1;
    }

Each job will be uniquely identified by an job Id which in this implementation will be a `UUID`. This allows jobs to be uniquely identified across any number of hosts.

The `Stop` call will terminate the given job using the `os.Process.Kill()` method which sends a `SIGKILL` signal to the underlying process. To ensure that any child processes are also terminated the `Pdeathsig` property will be set to `SIGKILL`.

The `Status` call returns the status of the given job.

    message StatusResponse {
        enum StatusType {
            RUNNING = 0;
            COMPLETED = 1;
            STOPPED = 2;
    	}
    	StatusType status = 1;
    	int32 exitCode = 2;
    }
    
The status of a job may be either running, completed or stopped. If the job has completed, the exit code is also populated otherwise it takes the default value of zero.

    message Log {
    	string logLine = 1;
    }

Logs are simply composed of log lines which are streamed to the client one at a time. In terms of performance it may be more efficient, depending on the deployment context, to stream the log lines in larger batches but this implementation aims for the simplest approach. The `GetLogs` call behaves like `tail -f -n +1` in that it will stream the output of the job from the beginning and will continue to stream until the job is finished. After the job has completed `GetLogs` will return the whole output.

## Library
The core functionality of the service is provided by the library functions. These allow clients to submit jobs, query the status of jobs, stop jobs and stream the logs from jobs.

Internally the server will use a standard map, protected by a `RWMutex`, to store a mapping from a `UUID` to a struct representing the job. An entry is inserted into the map for each job started on the server and can be subsequently queried using the `UUID`.

Jobs will be run using the `os/exec` package as this allows for running external processes and capturing their output. Both the `stdout` and `stderr` streams of the job will be captured to a buffer which will use `sync.RWLock` to enable multiple readers to read the output while it is being written.

## Client
A simple command line client is included to give an example of how this library could be used by other client applications. The following examples demonstrate its usage.

    $ ./client -c find /
    Submitted job id: 568
    
    $ ./client -s 568
    Status: running
    
    $ ./client -k 568
    
    $ ./client -s 568
    Status: completed
    Exit code: -1
    
    $ ./client -l 568
    /
    /home
    /home/thompsy
    /home/thompsy/.bash_history
    /home/thompsy/go
    /home/thompsy/go/bin
	... snip ...


## Security
### Authentication
Authentication will be provided by `mTLS` using TLS v1.3 which provides both performance benefits over TLS v1.2, due to fewer round trips needed to complete the handshake, and also security enhancements since it removes cipher suites with known vulnerabilities. This also removes the necessity of choosing a cipher suite since Go does not permit setting a specific cipher suite with TLS v1.3. The server will require clients to use v1.3 and will not complete the handshake for clients attempting to use v1.2. 

In a production environment where the clients could be run by different organisations the strict choice of TLS v1.3 may not be practical but in this instance, since both client and server are under our control, it is wise to enforce the strongest security possible. If TLS v1.2 had to be supported, using a strong cipher suite which provides forward secrecy like `TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384` would provide suitable security.

The use of `mTLS` requires the use of a number of certificates and keys. The server, along with each client, will have their own private key along with a certificate which has been signed by a trusted Certificate Authority. The certificates will be tied to a specific hostname and IP address. Upon connecting both client and server will verify that the other party is using a certificate signed by a trusted CA and also that the hostname and IP address details match.

All required keys and certificates, including those for the Certificate Authority, will be generated automatically as part of the build process using the `openssl` tool. In a production environment it would be important to store these securely outside of the standard public repository. A solution like Hashicorp's Vault or AWS Certificate Manager could be used.

### Authorization
The server will use a basic Role Based Access Control scheme to limit access to jobs. In this scheme there will be a single role: Process Owner. Process Owners will have permission to start new jobs and run any operation on jobs that they have started but will not have visibility or permission to modify jobs started by other clients.

In order to determine the identity of clients an email will be used as the `CommonName` of the client certificates. Since these certificates will be signed by a trusted Certificate Authority we can have confidence that this email correctly identifies the client. This email will also be used to determine group ownership. When a new job is started the email of the client is stored with the job to prevent other clients accessing it.

### Build Process
A simple `Makefile` will be provided to allow for easy and reproducible builds. This will include the generation of all required certificates along with static analysis of the code.

### Out of Scope

If the system was to be productionized, there are a number of additional features which it would be important to implement. These would include:

* expiring jobs. Currently all jobs along with their output are kept in memory by the server for its lifetime. In a production system this would eventually lead the server to run out of memory. A number of sensible schemes could be used to remove jobs e.g. once their output has been fetched by a client or after a set time period.

* persisting the output of jobs. As noted above, the output of a job is stored in memory. When the server exits all data is lost. A production system would likely want to persist this data to an external data store such as PostgreSQL.

* preventing unsafe jobs. This implementation provides no safety checking on commands submitted. This means that clients can submit jobs that perform destructive operations on the server e.g. `rm -rf`. A production system should either perform some safety checking on user input or provide an isolated environment e.g. using `cgroups` and `namespaces` directly or an existing Docker image.

* limiting the run-time of jobs. Jobs submitted currently have no timeout and may therefore run indefinitely or until stopped by the user e.g. using the `sleep` command. In a production system it would be sensible for the server to proactively kill jobs after a given period.

* resource limiting. In its current state the server imposes no resource limits on jobs. If clients were to submit many resource intensive jobs this could exhaust the hosts resources. To improve on this, resource limits could be enabled using `cgroups` and adding jobs to a resource constrained group.

* rate limiting the submission of jobs. This implementation makes no attempt to limit the number of jobs submitted either globally or on a per-client basis. A malicious or negligent client could use this fact to effect a denial of service attack against the server.

* accepting user input. The server will not supply any input to the commands as `stdin` will be connected to `/dev/null`. A future improvement could enable the server to accept a string of input from the client when the job is submitted or could allow the client to stream any required input as needed.

* performance metrics. The server will not generate any metrics. In a production environment this would be an important addition and could easily be added using appropriate tools like Prometheus and Grafana.

* high availability. As is, the server is a single process on a single machine and is therefore not resilient or highly available. Packaging the server into a container for deployment on a Kubernetes cluster would be a sensible option so that a number of pods could serve a particular ingress point. An important caveat is that level 7 routing would need to be setup so that requests related to a particular job were routed to the pod on which that job ran. 
