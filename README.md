# Polyscale Metrics
This is a testing app that executes a query:
- from every Section location to which it is deployed,
- into a [Polyscale.ai](https://polyscale.ai) cache,
- and then in-turn into an origin database for any cache misses.
By default the query executes every 60 seconds and emits a log entry, which we'll show below. The app also exposes a metrics endpoint that is scraped by a Grafana agent, and those metrics are then sent to Grafana Cloud. The metrics are the p50, p90, and p95 latencies over time.

There is no need to build the image, we provide one for you on https://ghcr.io/section/polyscale-metrics. The deployment yaml refers to that one. You just need to substitute your secrets and connection strings and deploy the yamls to your Section Project.

## Database Setup
You'll first need to have a database of some kind, one that is supported by Polyscale. [Supabase](https://supabase.com) is a managed Postgres database we've used before, but any Postgres database will suffice. And then you'll need to setup a cache at [Polyscale.ai](https://polyscale.ai), a managed service with a distributed global database cache.

## Deployment to Section

### Grafana Agent
First deploy the ConfigMap. Replace items in the remote_write section:

```
    remote_write:
    - url: GRAFANA_METRICS_INSTANCE_REMOTE_WRITE_ENDPOINT
      basic_auth:
        username: GRAFANA_METRICS_INSTANCE_ID
        password: GRAFANA_API_KEY
```

And apply:

```
$ kubectl apply -f grafana-app-scrape-configmap.yaml
```

In the Deployment resource no substitutions required. So just apply:

```
$ kubectl apply -f grafana-app-agent-deployment.yaml
```

### Polyscale Metrics App
No substitutions are required in the Service resource, so just apply:

```
$ kubectl apply -f polyscale-metrics-service.yaml
```

Substitute YOUR_CACHE_CONNECTION_STRING, YOUR_ORIGIN_CONNECTION_STRING, and YOUR_QUERY in the Deployment resource and apply:

```
$ kubectl apply -f polyscale-metrics-deployment.yaml
```

## Checking the Logs
In the following example I'm using a location optimizer config that has my Section Project running in 5 locations around the world.

```
$ kubectl --kubeconfig kubeconfig get pods -o wide
NAME                                 READY   STATUS    RESTARTS   AGE     IP               NODE        NOMINATED NODE   READINESS GATES
grafana-app-agent-66d6fffdf-78xdf    1/1     Running   0          36m     10.244.49.34     atl-bgaun   <none>           <none>
grafana-app-agent-66d6fffdf-fnc7w    1/1     Running   0          36m     10.244.23.244    sin-dhl6s   <none>           <none>
grafana-app-agent-66d6fffdf-xqjvd    1/1     Running   0          36m     10.244.56.43     rio-kz6s3   <none>           <none>
grafana-app-agent-66d6fffdf-xtt8x    1/1     Running   0          36m     10.247.188.9     syd-3pl5y   <none>           <none>
grafana-app-agent-66d6fffdf-zrqr9    1/1     Running   0          36m     10.244.58.242    sof-zzbd7   <none>           <none>
polyscale-metrics-77555d94b8-96772   1/1     Running   0          2m18s   10.244.58.235    atl-bgaun   <none>           <none>
polyscale-metrics-77555d94b8-w8fqr   1/1     Running   0          2m18s   10.244.70.130    sof-zzbd7   <none>           <none>
polyscale-metrics-77555d94b8-2gkf9   1/1     Running   0          2m19s   10.244.23.249    sin-dhl6s   <none>           <none>
polyscale-metrics-77555d94b8-sklh4   1/1     Running   0          2m18s   10.244.71.98     syd-3pl5y   <none>           <none>
polyscale-metrics-77555d94b8-8w5mg   1/1     Running   0          2m18s   10.244.120.117   rio-kz6s3   <none>           <none>
```

Looking at the logs from one of those pods shows great cache response as compared to the origin.

```
$ kubectl --kubeconfig kubeconfig logs polyscale-metrics-77555d94b8-96772
Node lmn-atl-k1-shared-ingress5 listening to /metrics on port :2112 interval 60 s query select * from foodsales limit 1;
nodename lmn-atl-k1-shared-ingress5 cache 241ms origin 416ms 
nodename lmn-atl-k1-shared-ingress5 cache 11ms origin 207ms 
nodename lmn-atl-k1-shared-ingress5 cache 11ms origin 208ms 
```