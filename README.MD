
<p align="center">
  <img height="200px" src="assets/logo.png">
  <br>
  <br>
  <bold>Managing Kubernetes Quota with confidence</bold>
</p>

# Kotary
![GitHub Workflow Status](https://img.shields.io/github/workflow/status/ca-gip/kotary/Build)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/ca-gip/kotary)
[![Go Report Card](https://goreportcard.com/badge/github.com/ca-gip/kotary)](https://goreportcard.com/report/github.com/ca-gip/kotary)
![Docker Pulls](https://img.shields.io/docker/pulls/cagip/kotary)

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Kotary](#kotary)
  - [What is it ?](#what-is-it-)
  - [Why not use an admission controller ?](#why-not-use-an-admission-controller-)
  - [Adding or Scaling-Up a Quota](#adding-or-scaling-up-a-quota)
  - [Installation](#installation)
      - [Add the CRD](#add-the-crd)
      - [Configuration](#configuration)
        - [Options](#options)
        - [Example](#example)
      - [Deployment](#deployment)
        - [Deploy the controller](#deploy-the-controller)
        - [(Optional) Deploy the service monitor](#optional-deploy-the-service-monitor)
  - [Getting Started](#getting-started)
    - [Update a ResourceQuota](#update-a-resourcequota)
      - [Example](#example-1)
      - [Status](#status)
        - [Example of a rejected claim](#example-of-a-rejected-claim)
        - [Example of a pending claim](#example-of-a-pending-claim)
    - [Default claim](#default-claim)
  - [Plan](#plan)
  - [Manage](#manage)
    - [Global](#global)
    - [Namespaces](#namespaces)
    - [Namespace Details](#namespace-details)
  - [Versioning](#versioning)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## What is it ?

It is an operator that brings a layer of verification and policy to the native _ResourceQuotas_ mechanism.
It introduced a new resource call a _ResourceQuotaClaims_ that will let users ask to modify the specification of their quota.
The verification includes :
* There are enough resources (CPU and Memory) on the cluster to allocate the claim which will look at the total amount
of resources of the cluster (worker node) and the sum of all the other _ResourceQuotas_ 
* _(Optional)_ It respects the maximum bound value express as a ratio of the cluster resource,
ex: a namespace cannot claim more that a 1/3 of the cluster
* _(Optional)_ In order to have some flexibility it is possible to set an over-commit or under-commit ratio to set what
is claimable compared to the actual resources. 
ex: In a development environment you could choose to allow reserving more resources than what is actually usable in reality.

In order to facilitate the adaption of _ResourceQuotaClaims_ it is possible to enforce a default claim for namespaces.
The feature will be activated on namespace that contains the label __quota=managed__.

## Why not use an admission controller ?

It could have been an elegant solution to use the admission controller mechanism in Kubernetes. This would have avoided the use 
of a Custom Resource Definition by directly asking to modify a _ResourceQuotas_. In the meantime this would have left out users on 
managed cluster like EKS, AKS or GKE, this is why we implemented the operator pattern instead.

## Adding or Scaling-Up a Quota

![How Kotary verification process work when you add a claim or request a scale-up](https://i.imgur.com/r0nIXl5.gif)

## Installation

#### Add the CRD
```bash
kubectl apply -f https://raw.githubusercontent.com/ca-gip/kotary/master/artifacts/crd.yml
```

#### Configuration

##### Options

| Name                           | Description                                                | Mandatory   | Type           | Default                  |
| :--------------                | :--------------------------------------------------------: | :---------:  | :-------------: |:---------------------- |
|  **defaultClaimSpec**          |  *Default claim that will be added to a watched Namespace* | `no`        | `ResourceList` | cpu:2 <br /> memory: 6Gi |
|  **ratioMaxAllocationMemory**  |  *Maximum amount of Memory claimable by a Namespace*       | `no`        | `Float`        | 1                        |
|  **ratioMaxAllocationCPU**     |  *Maximum amount of CPU claimable by a Namespace*          | `no`        | `Float`        | 1                        |
|  **ratioOverCommitMemory**     |  *Memory over-commitment*                                  | `no`        | `Float`        | 1                        |
|  **ratioOverCommitCPU**        |  *CPU over-commitment*                                     | `no`        | `Float`        | 1                        |

##### Example

In the following sample configuration we set :
 * A default claim of 2 CPU and 10Gi of Memory
 * 33% of total amount of resource can be claim by a namespace
 * An over-commit of 130%

```bash
cat <<EOF | kubectl -n kube-system create -f -
apiVersion: v1
kind: ConfigMap
data:
  defaultClaimSpec: |
    cpu: "2"
    memory: "10Gi"
  ratioMaxAllocationMemory: "0.33"
  ratioMaxAllocationCPU: "0.33"
  ratioOverCommitMemory: "1.3"
  ratioOverCommitCPU: "1.3"
metadata:
  name: kotary-config
EOF
```

#### Deployment

##### Deploy the controller

```bash
kubectl apply -f https://raw.githubusercontent.com/ca-gip/kotary/master/artifacts/deployment.yml
```

##### (Optional) Deploy the service monitor 

```bash
kubectl apply -f https://raw.githubusercontent.com/ca-gip/kotary/master/artifacts/metrics.yml
```

# Contributing

- [Contributing to kotary](https://github.com/ca-gip/kotary/blob/master/CONTRIBUTING.md).
 

 <hr/>
<p align=center  style="background-color:#333333 !important;">
  <a href="https://www.jetbrains.com/">
  Developed with
  <br/>
  <img align="center" src="https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.png" alt="drawing" width="100"/>
  </a>
</p>
