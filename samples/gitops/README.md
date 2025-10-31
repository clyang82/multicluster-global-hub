# Simple GitOps with OpenShift GitOps and ACM Global Hub

This sample shows how to use OpenShift GitOps to deploy a simple application through ACM Global Hub.

## Use Cases

### Use Case 1: Deploy to Local Cluster (Global Hub itself)
- Folder: `local-cluster/`
- Deploys directly to the Global Hub cluster
- Good for: Global Hub management applications, monitoring tools

### Use Case 2: Deploy to Remote Managed Clusters
- Folder: `remote-clusters/`
- Deploys through ACM to managed clusters
- Good for: Applications, policies across your fleet

## Quick Setup

### For Local Cluster Deployment:
```bash
oc apply -f local-cluster/argocd-app.yaml
```

### For Remote Clusters Deployment:
```bash
oc apply -f remote-clusters/argocd-app.yaml
```

## Results

- ArgoCD UI: See the sync status
- ACM Console: See application status
- Global Hub Dashboard: See aggregated status across hubs