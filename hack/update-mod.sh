#!/bin/bash
# This script is not used, but as been useful for debugging.
go mod edit --replace k8s.io/api=k8s.io/api@v0.24.13
go mod edit --replace k8s.io/apiextensions-apiserver=k8s.io/apiextensions-apiserver@v0.24.13
go mod edit --replace k8s.io/apimachinery=k8s.io/apimachinery@v0.24.13
go mod edit --replace k8s.io/apiserver=k8s.io/apiserver@v0.24.13
go mod edit --replace k8s.io/cli-runtime=k8s.io/cli-runtime@v0.24.13
go mod edit --replace k9s.io/client-go=k8s.io/cli-runtime@v0.24.13
go mod edit --replace k8s.io/cloud-provider=k8s.io/cloud-provider@v0.24.13
go mod edit --replace k8s.io/cluster-bootstrap=k8s.io/cluster-bootstrap@v0.24.13
go mod edit --replace k8s.io/code-generator=k8s.io/code-generator@v0.24.13
go mod edit --replace k8s.io/component-base=k8s.io/component-base@v0.24.13
go mod edit --replace k8s.io/cri-api=k8s.io/cri-api@v0.24.13
go mod edit --replace k8s.io/csi-translation-lib=k8s.io/csi-translation-lib@v0.24.13
go mod edit --replace k8s.io/kube-aggregator=k8s.io/kube-aggregator@v0.24.13
go mod edit --replace k8s.io/kube-controller-manager=k8s.io/kube-controller-manag@v0.24.13
go mod edit --replace k8s.io/kube-proxy=k8s.io/kube-proxy@v0.24.13
go mod edit --replace k8s.io/kube-scheduler=k8s.io/kube-scheduler@v0.24.13
go mod edit --replace k8s.io/kubelet=k8s.io/kubelet@v0.24.13
go mod edit --replace k8s.io/legacy-cloud-providers=k8s.io/legacy-cloud-providers@v0.24.13
go mod edit --replace k8s.io/metrics=k8s.io/metrics@v0.24.13
go mod edit --replace k8s.io/sample-apiserver=k8s.io/sample-apiserver@v0.24.13