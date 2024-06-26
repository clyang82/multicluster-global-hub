# Event API Specifications
These are events generated on the Kafka topics by multicluster global hub. The events are for the policy and managed cluster right now. We may extend to support other types in the future.

## Topics
The following Kafka topics are used
- status.$(managed_hub_cluster_name)
- event

## Event Structure

The events are formatted as per [CloudEvents](https://github.com/cloudevents/spec/blob/main/cloudevents/spec.md) Specifications. Therefore the message envelopes are common as specified by the CloudEvent Specs. It is encoded in JSON format.

A simple `hello world` message would look like.  - 
```
{
    "specversion" : "1.0",
    "type" : "com.event",
    "source" : "source",
    "id" : "A234-1234-1234",
    "time" : "2018-04-05T17:31:00Z",
    "datacontenttype" : "application/json",
    "data" : {
        "message" : "Hello World!"
    }
}
```

_CloudEvent attributes are prefixed with ce_ for use in the message-headers section in Kafka._

Examples:

* `time` maps to `ce-time`
* `id` maps to `ce-id`
* `specversion` maps to `ce-specversion`

## Topic: status.$(managed_hub_cluster_name)
### Events related to Policy
#### Local Policy Spec
The event includes the policy spec which is applied in the managed hub cluster. The `source` specifies the managed hub cluster name. The `data` is for the policy spec. The events are always sent by the hub cluster.
```
{
  "specversion": "1.0",
  "id": "7923be59-0050-4112-90c0-1e8d1ec4d486",
  "source": "kind-hub1",
  "type": "io.open-cluster-management.operator.multiclusterglobalhubs.policy.localspec",
  "datacontenttype": "application/json",
  "time": "2024-02-29T03:01:12.717129594Z",
  "data": [
    {
      "kind": "Policy",
      "apiVersion": "policy.open-cluster-management.io/v1",
      "metadata": {
        "name": "policy-limitrange",
        "namespace": "local-policy-namespace",
        "uid": "cd0a3ae3-6579-4fbc-b9c1-eb7e9d15dd6d",
        "resourceVersion": "849216",
        "creationTimestamp": "2024-02-29T03:01:04Z",
        "annotations": {
          "policy.open-cluster-management.io/categories": "PR.IP Information Protection Processes and Procedures",
          "policy.open-cluster-management.io/controls": "PR.IP-1 Baseline Configuration",
          "policy.open-cluster-management.io/standards": "NIST-CSF"
        }
      },
      "spec": {
        "disabled": false,
        "remediationAction": "inform",
        "policy-templates": [
          {
            "objectDefinition": {
              "apiVersion": "policy.open-cluster-management.io/v1",
              "kind": "ConfigurationPolicy",
              "metadata": {
                "name": "policy-limitrange-container-mem-limit-range"
              },
              "spec": {
                "namespaceSelector": {
                  "exclude": [
                    "kube-*"
                  ],
                  "include": [
                    "default"
                  ]
                },
                "object-templates": [
                  {
                    "complianceType": "musthave",
                    "objectDefinition": {
                      "apiVersion": "v1",
                      "kind": "LimitRange",
                      "metadata": {
                        "name": "container-mem-limit-range"
                      },
                      "spec": {
                        "limits": [
                          {
                            "default": {
                              "memory": "512Mi"
                            },
                            "defaultRequest": {
                              "memory": "256Mi"
                            },
                            "type": "Container"
                          }
                        ]
                      }
                    }
                  }
                ],
                "remediationAction": "inform",
                "severity": "medium"
              }
            }
          }
        ]
      },
      "status": {}
    }
  ]
}
```
#### Local Complete Policy Compliance Status
The event includes the policy compliance status. It is a complete status. The `source` specifies the managed hub cluster name. The `data` is for the policy compliance status, including compliant clusters, non-compliant clusters and unknown clusters.

```
{
  "specversion": "1.0",
  "id": "d6561516-c8d5-4c3f-98e3-df650ee8e809",
  "source": "kind-hub1",
  "type": "io.open-cluster-management.operator.multiclusterglobalhubs.policy.localcompletecompliance",
  "datacontenttype": "application/json",
  "time": "2024-02-29T03:01:12.624601184Z",
  "data": [
    {
      "policyId": "cd0a3ae3-6579-4fbc-b9c1-eb7e9d15dd6d",
      "nonCompliantClusters": [
        "kind-hub1-cluster1"
      ],
      "compliantClusters": [
        "kind-hub1-cluster2"
      ],
      "pendingClusters":[

      ],
      "unknownComplianceClusters": []
    }
  ],
}
```
#### Local Policy Compliance Status
The event includes the policy compliance status. It is a delta status. The `source` specifies the managed hub cluster name. The `data` is for the policy compliance status. We merge the status with the existing status in the database.

```
{
  "specversion": "1.0",
  "id": "4b3e7f44-289c-4c67-b1d6-8e5d4b59ecbd",
  "source": "kind-hub1",
  "type": "io.open-cluster-management.operator.multiclusterglobalhubs.policy.localcompliance",
  "datacontenttype": "application/json",
  "time": "2024-02-29T03:01:07.625306267Z",
  "data": [
    {
      "policyId": "cd0a3ae3-6579-4fbc-b9c1-eb7e9d15dd6d",
      "compliantClusters": [],
      "nonCompliantClusters": [],
      "unknownComplianceClusters": [
        "kind-hub1-cluster1"
      ]
    }
  ]
}

```

### Events related to Cluster
#### Managed Cluster Spec and Status
This event includes the spec and status of the managed cluster. Once the managed cluster spec or status is changed, the whole information will be reported back.
```
{
  "specversion": "1.0",
  "id": "213d0259-8999-46bf-aa13-bf4a3684075f",
  "source": "kind-hub1",
  "type": "io.open-cluster-management.operator.multiclusterglobalhubs.spec.managedcluster",
  "datacontenttype": "application/json",
  "time": "2024-02-29T02:53:32.621450791Z",
  "data": [
    {
      "kind": "ManagedCluster",
      "apiVersion": "cluster.open-cluster-management.io/v1",
      "metadata": {
        "name": "kind-hub1-cluster1",
        "uid": "f6871dc9-1052-4a73-ba82-0db30bc41e89",
        "resourceVersion": "843851",
        "creationTimestamp": "2024-02-27T01:48:31Z",
        "labels": {
          "cluster.open-cluster-management.io/clusterset": "default",
          "feature.open-cluster-management.io/addon-application-manager": "available"
        },
        "annotations": {
          "global-hub.open-cluster-management.io/managed-by": "kind-hub1"
        }
      },
      "spec": {
        "managedClusterClientConfigs": [
          {
            "url": "https://hub1-cluster1-control-plane:6443",
            "caBundle": "XXX"
          }
        ],
        "hubAcceptsClient": true,
        "leaseDurationSeconds": 60
      },
      "status": {
        "conditions": [
          {
            "type": "HubAcceptedManagedCluster",
            "status": "True",
            "lastTransitionTime": "2024-02-27T01:48:32Z",
            "reason": "HubClusterAdminAccepted",
            "message": "Accepted by hub cluster admin"
          },
          {
            "type": "ManagedClusterJoined",
            "status": "True",
            "lastTransitionTime": "2024-02-27T01:48:32Z",
            "reason": "ManagedClusterJoined",
            "message": "Managed cluster joined"
          },
          {
            "type": "ManagedClusterConditionAvailable",
            "status": "True",
            "lastTransitionTime": "2024-02-27T01:48:32Z",
            "reason": "ManagedClusterAvailable",
            "message": "Managed cluster is available"
          }
        ],
        "capacity": {
          "cpu": "8",
          "ephemeral-storage": "83156972Ki",
          "hugepages-1Gi": "0",
          "hugepages-2Mi": "0",
          "memory": "32232364Ki",
          "pods": "110"
        },
        "allocatable": {
          "cpu": "8",
          "ephemeral-storage": "83156972Ki",
          "hugepages-1Gi": "0",
          "hugepages-2Mi": "0",
          "memory": "32232364Ki",
          "pods": "110"
        },
        "version": {
          "kubernetes": "v1.24.0"
        },
        "clusterClaims": [
          {
            "name": "id.k8s.io",
            "value": "893013f7-90cf-4d44-b598-32437e4fca3c"
          }
        ]
      }
    }
  ]
}
```
### ManagedHubCluster
It is designed to be used internally. The event reflects the managed hub cluster information and liveness.
#### Heartbeat
```
{
  "specversion": "1.0",
  "id": "1f062142-0ab5-43d3-b428-171e323f8a49",
  "source": "kind-hub1",
  "type": "io.open-cluster-management.operator.multiclusterglobalhubs.managedhub.heartbeat",
  "datacontenttype": "application/json",
  "time": "2024-02-29T02:57:27.627736066Z",
  "data": []
}
```
### HubClusterInfo
```
{
  "specversion": "1.0",
  "id": "11661511-979c-4e0c-810e-a62b5ddb11c1",
  "source": "kind-hub1",
  "type": "io.open-cluster-management.operator.multiclusterglobalhubs.managedhub.info",
  "datacontenttype": "application/json",
  "time": "2024-02-29T02:54:27.627870336Z",
  "data": {
    "consoleURL": "https://console-openshift-console.apps.xxx.com",
    "grafanaURL": "",
    "clusterId": "8606cf88-c711-4dae-ba7e-2e4d54227887"
  }
}
```

## Topic: event
Currently, the following resource events are supported:
- Policy
- Cluster

The event is a Kubernetes event in the managed hub clusters or managed clusters.

### Events related to Policy
The events are Kubernetes events. We collect the root policy events and replicated policy events.
#### Root Policy Event
The event is Kubernetes event that exists in root policy namespace. The event specifies which clusters are applied for this policy.
```
{
  "specversion": "1.0",
  "id": "4e85317c-9208-4c9d-be22-4a2725867670",
  "source": "kind-hub2",
  "type": "io.open-cluster-management.operator.multiclusterglobalhubs.event.localrootpolicy",
  "datacontenttype": "application/json",
  "time": "2024-02-29T03:01:06.30007874Z",
  "data": [
    {
      "leafHubName": "kind-hub2",
      "eventName": "policy-limitrange.17b8363660d39188",
      "eventNamespace": "local-policy-namespace",
      "message": "Policy local-policy-namespace/policy-limitrange was propagated to cluster kind-hub2-cluster1/kind-hub2-cluster1",
      "reason": "PolicyPropagation",
      "count": 1,
      "source": {
        "component": "policy-propagator"
      },
      "createdAt": "2024-02-29T03:01:05Z",
      "policyId": "1f2deb7a-0d29-4762-b0fc-daa3ba16c5b5",
      "compliance": "Unknown"
    }
  ]
}
```
#### Replicated Policy Event
The event is from the replicated policy history status. The event is Kubernetes event and exists in the managed cluster.
```
{
  "specversion": "1.0",
  "id": "5b5917b5-1fa2-4eb8-a7fa-c1d97dc96218",
  "source": "kind-hub2",
  "type": "io.open-cluster-management.operator.multiclusterglobalhubs.event.localreplicatedpolicy.update",
  "datacontenttype": "application/json",
  "time": "2024-02-29T03:01:16.387894285Z",
  "data": [
    {
      "leafHubName": "kind-hub2",
      "eventName": "local-policy-namespace.policy-limitrange.17b83638614ff6b7",
      "eventNamespace": "kind-hub2-cluster1",
      "message": "NonCompliant; violation - limitranges [container-mem-limit-range] not found in namespace default",
      "reason": "PolicyStatusSync",
      "count": 1,
      "source": {
        "component": "policy-status-history-sync"
      },
      "createdAt": "2024-02-29T03:01:14Z",
      "policyId": "1f2deb7a-0d29-4762-b0fc-daa3ba16c5b5",
      "clusterId": "0884ef05-115d-46f5-bbda-f759adcbbe5b",
      "clusterName": "kind-hub2-cluster1",
      "compliance": "NonCompliant"
    }
  ]
}
```

### Events related to Cluster
The events are Kubernetes events. We collect the cluster life cycle events, including import and detach.
#### Import
The events are related to cluster import. The data is an array that can include multiple events together. Once the cluster is successfully imported, it is managed by ACM.
```
{
    "specversion": "1.0",
    "id": "c006d9b2-bc2d-11ee-90e9-8feab4dd9214",
    "source": "managed_hub1",
    "type": "io.open-cluster-management.operator.multiclusterglobalhubs.event.managedcluster",
    "datacontenttype": "application/json",
    "time": "2024-01-25T04:06:16.542703828Z",
    "data": [
        {
          "leafHubName": "managed_hub1",
          "clusterId": "6b9b8545-1a84-4b55-8423-a9b28a1a4967",
          "clusterName": "cluster1",
          "eventName": "kube-system.import.17ad7b80d4e6f6a4",
          "message": "The cluster is being imported now",
          "reason": "Importing",
          "count": 1,
          "source": "import-controller",
          "createdAt": "2024-01-25T05:08:07Z"
        },
        {
          "leafHubName": "managed_hub1",
          "clusterId": "6b9b8545-1a84-4b55-8423-a9b28a1a4967",
          "clusterName": "cluster1",
          "eventName": "kube-system.import.17ad7b80d4ade6a4",
          "message": "The cluster is imported successfully",
          "reason": "Imported",
          "count": 1,
          "source": "import-controller",
          "createdAt": "2024-01-25T05:08:17Z"
        }
    ]
}
```
#### Detach
The events are related to cluster detach. We do not have `Detached` event. However, we can always treat the detaching process as successful anyway.
```
{
    "specversion": "1.0",
    "id": "a59375e4-bc2e-11ee-98bb-035b5cb373d3",
    "source": "managed_hub1",
    "type": "io.open-cluster-management.operator.multiclusterglobalhubs.event.managedcluster",
    "datacontenttype": "application/json",
    "time": "2024-01-25T04:06:16.542703828Z",
    "data": [
        {
          "leafHubName": "managed_hub1",
          "clusterId": "6b9b8545-1a84-4b55-8423-a9b28a1a4967",
          "clusterName": "cluster1",
          "eventName": "kube-system.status.17ad7b80d4e6f6a4",
          "message": "The cluster (cluster1) is being detached now",
          "reason": "Detaching",
          "count": 1,
          "source": "import-controller",
          "createdAt": "2024-01-25T05:08:07Z"
        } 
    ]
}
```