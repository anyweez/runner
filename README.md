Runner
======

Runner is a job retrieval tool that connects to a [beanstalkd](http://kr.github.io/beanstalkd/) instance and retrieves jobs to be
executed on the local machine. It uses a local mapping between job names and actual executables
as a layer of abstraction to prevent arbitary (and potentially dangerous) jobs to come in off of
the queue.

Runner is designed to be a very simple tool with speed, simplicity, and security as core design principles. It
can execute any kind of job that the local machine can execute, has parameterized inputs to
control for the number of simultaneous processes, and only requires configuration via simple flags.

This package also includes a protobuffer definition that all jobs requests are expected to arrive in, but can
be replaced fairly easily.

Available flags:
* source *(required)*: the network address of the beanstalkd instance that jobs should be pulled from (ip or ip:port)
* target\_queue *(optional)*: the name of the beanstalkd tube that jobs should be pulled from, if any
* limit *(optional, default: 2)*: the number of simultaneous jobs that should be executed simultaneously	
* command\_file *(optional, default: data/commands.json)*: the file describing the commands that can be run

Another binary, emit, is also included that can be used to easily send job requests to the queue using the provided proto.
Run `emit --help` to see the (relatively simple) options for that tool. Note that you don't need to use emit in order to
take advantage of runner; it's merely included as a convenience.	
