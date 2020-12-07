# About AppShare Operator
This operator is very much work in progress. The operator will install AppShare (https://github.com/zubairq/appshare) on OpenShift, and possibly Kubernetes as well, but that is still to be tested.

## Prerequisites
- Golang v. 1.13 or greater
- Operator SDK (https://sdk.operatorframework.io/)

## How to test
1. Make sure you are logged in to a cluster
2. Build a docker image and push it to a registry:<br>`make docker-build docker-push IMG=<some-registry>/<project-name>:<some-tag>`
3. Install and deploy:<br>`make install`<br>`make deploy IMG=<some-registry>/<project-name>:<some-tag>`
4. Create the CRD. See `config/samples/appshare_v1_appshare.yaml`
