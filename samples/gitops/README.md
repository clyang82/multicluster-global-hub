# Simple GitOps with OpenShift GitOps and ACM Global Hub

This sample shows how to use OpenShift GitOps (ArgoCD) to deploy applications through ACM Global Hub.

## Use Cases

### Use Case 1: Deploy to Local Cluster (Global Hub itself)
- Folder: `local-cluster/`
- ArgoCD creates ACM resources targeting local-cluster
- Good for: Global Hub management applications, monitoring tools

### Use Case 2: Deploy to Remote Managed Clusters
- Folder: `remote-clusters/`
- ArgoCD creates ACM resources targeting remote clusters
- Good for: Applications, policies across your fleet

Both use cases use the same ACM pattern (Channel/Subscription/Placement) with different placement targeting.

## Quick Setup

### For Local Cluster Deployment:
```bash
oc apply -f local-cluster/argocd-app.yaml
```
This creates ACM resources that deploy nginx to the local-cluster.

### For Remote Clusters Deployment:
```bash
oc apply -f remote-clusters/argocd-app.yaml
```
This creates ACM resources that deploy nginx to remote managed clusters.

## Results

- ArgoCD UI: See the sync status
- ACM Console: See application status
- Global Hub Dashboard: See aggregated status across hubs