# aws-lambda-go
========

An example of running [AWS Lambda](https://aws.amazon.com/lambda) functions in Go, using [eawsy/aws-lambda-go-shim](https://github.com/eawsy/aws-lambda-go-shim).

To try this example, simply setup your AWS environment with
```sh
aws configure
```
and get dependecy with
```sh
go get -u -d github.com/eawsy/aws-lambda-go-core/...
```
then run the `deploy.sh` script
```sh
./deploy.sh
```
