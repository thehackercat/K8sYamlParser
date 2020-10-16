# K8sYamlParser

[![Go Report Card](https://goreportcard.com/badge/github.com/thehackercat/K8sYamlParser)](https://goreportcard.com/report/github.com/thehackercat/K8sYamlParser)

translate k8s yaml to client-go code recognized spec

## Run

``` bash
go run main.go -kubeconfig ~/.kube/config -file tests/demo.yaml
```