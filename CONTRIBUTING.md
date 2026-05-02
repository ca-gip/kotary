
## Getting Started

### Update a ResourceQuota

To update a _ResourceQuotas_ you will have to create a _ResourceQuotaClaims_ with specification for CPU and Memory.
You can use the same units as the one available in Kubernetes, 
please refer to the [official documentation](https://kubernetes.io/docs/tasks/administer-cluster/manage-resources/quota-memory-cpu-namespace/)

#### Example

```bash
cat <<EOF | kubectl apply -n demo-ns -f -
apiVersion: cagip.github.com/v1
kind: ResourceQuotaClaim
metadata:
  name: demo
spec:
  memory: 20Gi
  cpu: 5
EOF
```

#### Status

After creating a _ResourceQuotaClaims_ there are three possibilities:
* __Accepted__ : The claim will be deleted, and the modifications are applied to the _ResourceQuota_
* __Rejected__ : It was not possible to accept the modification the claim show a status "REJECTED" with details.
* __Pending__ : The claim is requesting less resources than what is currently requested on the namespace, the claim will be accepted once it's possible to downscale

##### Example of a rejected claim

```bash
$ kubectl get quotaclaim
NAME   CPU   RAM    STATUS     DETAILS
demo   5     20Gi   REJECTED   Exceeded Memory allocation limit claiming 20Gi but limited to 18Gi
```

##### Example of a pending claim

```bash
$ kubectl get quotaclaim
NAME   CPU   RAM    STATUS     DETAILS
demo   5     16Gi   PENDING    Awaiting lower CPU consumption claiming 16Gi but current total of CPU request is 18Gi
```

### Default claim

If you are using the default claim policy, namespace will automatically receive a claim and if all the verifications 
pass a managed-quota will be applied.

```bash
$ kubectl get resourcequota
NAME            CREATED AT
managed-quota   2020-01-24T08:31:32Z
```

## Plan

Implementing _ResourceQuota_ when you already have running workload on your cluster can be a tedious task.
To help you get started you can use our cli [kotaplan](https://github.com/ca-gip/kotaplan). It will enable to test various scenarios beforehand
by simulating how the quota could be implemented according to the desired settings.

Here is a quick example

```bash
$ kotaplan -label quota=managed -cpuover 0.95 -memover 0.95 -memratio 0.33 -cpuratio 0.33
+---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
| Namespaces Details                                                                                                                                                                    |
+--------------+------+---------+-----------------+---------------+---------+-----------------+---------------+-------------+---------------------------+-------------------------------+
| NAMESPACE    | PODS | MEM REQ | CURRENT MEM USE | MEM REQ USAGE | CPU REQ | CURRENT CPU USE | CPU REQ USAGE | FIT DEFAULT | RESPECT MAX ALLOCATION NS | SPEC                          |
+--------------+------+---------+-----------------+---------------+---------+-----------------+---------------+-------------+---------------------------+-------------------------------+
| team-1-dev   |   14 | 7GiB    | 859.4MiB        | 11.9897 %     |    2800 |               6 | 0.2143 %      | false       | true                      | CPU : 3360m    MEM: 8.4GiB    |
| team-1-int   |   14 | 7GiB    | 852MiB          | 11.8868 %     |    2800 |               0 | 0 %           | false       | true                      | CPU : 3360m    MEM: 8.4GiB    |
| team-1-prd   |   16 | 8GiB    | 1.125GiB        | 14.0568 %     |    3200 |               0 | 0 %           | false       | true                      | CPU : 3840m    MEM: 9.6GiB    |
| team-2-dev   |   16 | 8GiB    | 1.119GiB        | 13.9935 %     |    3200 |               0 | 0 %           | false       | true                      | CPU : 3840m    MEM: 9.6GiB    |
| team-2-dev   |    8 | 4GiB    | 531.4MiB        | 12.9745 %     |    1600 |               0 | 0 %           | false       | true                      | CPU : 1920m    MEM: 6GiB      |
| team-2-dev   |   28 | 9GiB    | 1.033GiB        | 11.4748 %     |    3600 |               6 | 0.1667 %      | false       | true                      | CPU : 4320m    MEM: 10.8GiB   |
| team-3-dev   |    0 | 0B      | 0B              | 0 %           |       0 |               0 | 0 %           | true        | true                      | CPU : 1000m    MEM: 6GiB      |
| team-3-prd   |    0 | 0B      | 0B              | 0 %           |       0 |               0 | 0 %           | true        | true                      | CPU : 1000m    MEM: 6GiB      |
+--------------+------+---------+-----------------+---------------+---------+-----------------+---------------+-------------+---------------------------+-------------------------------+
| 8            |   96 |         |                 |               |         |                 |               |             |                           | CPU : 22692M    MEM: 64.8 GIB |
+--------------+------+---------+-----------------+---------------+---------+-----------------+---------------+-------------+---------------------------+-------------------------------+
+-------------------------------------------------------------+
| Summary                                                     |
+-------------------------------------------------------------+
| Number of nodes               8                             |
| Available resources (real)    CPU : 64000m    MEM: 250.5GiB |
| Available resources (commit)  CPU : 60800m    MEM: 238GiB   |
| Max per NS                    CPU : 21120m    MEM: 82.67GiB |
+-------------------------------------------------------------+
| RESULT                        OK                            |
+-------------------------------------------------------------+
```

## Manage

To help you manage effectively your _ResourceQuotas_ you can use the provided Granafa dashboard. You will be able
to set it up according to your configuration by modifying the dashboard variable.

### Global

The global section will enable users to check the current running configuration (manual) to size accordingly their claims.
It also shows what is currently available, reserved and claimable in terms of resources. 

![grafana_global](assets/grafana_global.png)


### Namespaces

This section list all the managed namespaces to give a rough idea of what is currently use by running containers. It allows to 
rapidly checks which quota should be increased or decreased.

![grafana_namespaces](assets/grafana_namespaces.png)

### Namespace Details

This section shows a detailed view of the Memory and CPU consumption on a particular namespace. It allows to visually check
what the specifications of the quota, the total amount of request made by containers and their real consumption.

![grafana_namespace_details](assets/grafana_namespace_details.png)

## Versioning 

Since version v1.24.0, we have decided to modify the naming of versions for ease of reading and understanding.
Example: v1.24.0 means that the operator was developed for Kubernetes version 1.24 and that the last 0 corresponds to the various patches we have made to the operator.


<!-- omit in toc -->
# Contributing to kotary

First off, thanks for taking the time to contribute! ❤️

All types of contributions are encouraged and valued. See the [Table of Contents](#table-of-contents) for different ways to help and details about how this project handles them. Please make sure to read the relevant section before making your contribution. It will make it a lot easier for us maintainers and smooth out the experience for all involved. The community looks forward to your contributions. 🎉

> And if you like the project, but just don't have time to contribute, that's fine. There are other easy ways to support the project and show your appreciation,   which we would also be very happy about:
> - Star the project
> - Tweet about it
> - Refer this project in your project's readme
> - Mention the project at local meetups and tell your friends/colleagues

<!-- omit in toc -->
## Table of Contents

- [I Have a Question](#i-have-a-question)
- [I Want To Contribute](#i-want-to-contribute)
- [Reporting Bugs](#reporting-bugs)
- [Suggesting Enhancements](#suggesting-enhancements)


## I Have a Question

> If you want to ask a question, we assume that you have reviewed the available [Documentation](https://github.com/ca-gip/kubi).


> Before asking, it is advisable to check existing Issues [Issues](https://github.com/ca-gip/kubi/issues) that may address your query. If you find a relevant issue but still require clarification, please post your question within that issue. Additionally, searching the internet for answers can often be helpful.

> Should you still need to ask a question after following these steps, we recommend:

- Sending an email to the mailing list CAGIP_DEVOPS_CONTAINER <cagip_devops_container@ca-gip.fr>.
- Providing as much context as possible regarding the issue you are encountering.
- Including relevant project and platform versions (e.g., Kubernetes, Golang) as applicable.
  
This approach ensures that your question reaches the right audience and is more likely to receive a prompt response.

Before you ask a question, it is best to search for existing  that might help you. In case you have found a suitable issue and still need clarification, you can write your question in this issue. It is also advisable to search the internet for answers first.


## I Want To Contribute

> ### Legal Notice <!-- omit in toc -->
> When contributing to this project, you must agree that you have authored 100% of the content, that you have the necessary rights to the content and that the content you contribute may be provided under the project license.

### Reporting Bugs

#### Before Submitting a Bug Report

A good bug report shouldn't leave others needing to chase you up for more information. Therefore, we ask you to investigate carefully, collect information and describe the issue in detail in your report. Please complete the following steps in advance to help us fix any potential bug as fast as possible.

- Make sure that you are using the latest version.
- Determine if your bug is really a bug and not an error on your side e.g. using incompatible environment components/versions (Make sure that you have read the [documentation](https://github.com/ca-gip/kubi). If you are looking for support, you might want to check [this section](#i-have-a-question)).
- To see if other users have experienced (and potentially already solved) the same issue you are having, check if there is not already a bug report existing for your bug or error.
- Collect information about the bug:
  - Stack trace (Traceback)
  - OS, Platform and Version (Windows, Linux, macOS, x86, ARM)
  - Version of the interpreter, compiler, SDK, runtime environment, package manager, depending on what seems relevant.
  - Possibly your input and the output
  - What did you have as result, and what did you expect ?
  - Can you reliably reproduce the issue? And can you also reproduce it with older versions?
  - Give everything we need to reproduce the issue (a test if possible)

<!-- omit in toc -->

Once it's filed:

- A team member will try to reproduce the issue with your provided steps and then we will contact you back.
- If the team is able to reproduce the issue, it will be marked `needs-fix`, as well as possibly other tags (such as `critical`), and the issue will be left to be [implemented by someone](#your-first-code-contribution).

<!-- You might want to create an issue template for bugs and errors that can be used as a guide and that defines the structure of the information to be included. If you do so, reference it here in the description. -->


### Suggesting Enhancements

This section guides you through submitting an enhancement suggestion for kotary, **including completely new features and minor improvements to existing functionality**. Following these guidelines will help maintainers and the community to understand your suggestion and find related suggestions.

<!-- omit in toc -->
#### How Do I Submit a Good Enhancement Suggestion?

To submit an enhancement suggestion, please propose a pull request (PR) and contact us for review.


 <hr/>
<p align=center  style="background-color:#333333 !important;">
  <a href="https://www.jetbrains.com/">
  Developed with
  <br/>
  <img align="center" src="https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.png" alt="drawing" width="100"/>
  </a>
</p>

