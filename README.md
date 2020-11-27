# An operator for AppShare

## How to test
1. Make sure you are logged in to a cluster
2. Build a docker image and push it to a registry:<br>`make docker-build docker-push IMG=<some-registry>/<project-name>:<some-tag>`
3. Install and deploy:<br>`make install`<br>`make deploy IMG=<some-registry>/<project-name>:<some-tag>`
4. Create the CRD. See `config/samples/appshare_v1_appshare.yaml`
