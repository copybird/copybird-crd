## Copybird k8s controller

This repository contains custom resource definition and its controller to create and run `Backup` objects in Kubernetes. Sample backup object available in [sample](https://github.com/copybird/copybird-crd/tree/master/samples) directory as well as test mysql pod and service yamls. Most convenient way to build and deploy this CRD is to use [ko](https://github.com/google/ko) tool which may be installed by running:

```   
GO111MODULE=on go get github.com/google/ko/cmd/ko
```

After installation is complete, you can simply run `ko apply -f config/` from the repository root and watch how all configurations and images being prepared for you. Please note that you must have k8s cluster configured in `$HOME/kube/config` (kubectl configuration).