# Polyscale Metrics

## Deployment to Section

### Grafana Configmap
Replace items in the remote_write section:

    remote_write:
    - url: GRAFANA_METRICS_INSTANCE_REMOTE_WRITE_ENDPOINT
      basic_auth:
        username: GRAFANA_METRICS_INSTANCE_ID
        password: GRAFANA_API_KEY

$ kubectl apply -f grafana-app-scrape-configmap.yaml

### Grafana Agent Deployment
No substitutions required.

$ kubectl apply -f grafana-app-agent-deployment.yaml

### Polyscale
No substitutions are required in the Service resource.

$ kubectl apply -f polyscale-metrics-service.yaml

Substitute YOUR_CACHE_CONNECTION_STRING, YOUR_ORIGIN_CONNECTION_STRING, and YOUR_QUERY in the Deployment resource.

$ kubectl apply -f polyscale-metrics-deployment.yaml
