# Server

The server is a simple thread that receives packets from the client, adds the timestamp and sends it back. It can be
deployed in all kind of environments provided that the client is able to reach it from inside or outside the LAN.

## How to deploy

```
docker pull richimarchi/latency-tester_server
docker run -p 8080:8080 [--name <container-name>] richimarchi/latency-tester_server [-addr=<ip:port>] [-tls=<enabled>]
```

Latest version: `1.0.0`

### Server flags:

|Param|Description|Default Value|
|---|---|---|
|`-addr`|Listening address and port|`0.0.0.0:8080`|
|`-tls`|`true` if TLS requested|`false`|

### How to deploy the server into a Kubernetes cluster

Starting from the `serverDeploymentSkeleton.yaml` file, generate the custom deployment file depending on your hostname and deploy it in your k8s cluster:

```
export HOSTNAME=<custom-hostname>
envsubst < serverDeploymentSkeleton.yaml > customServerDeployment.yaml
kubectl apply -f customServerDeployment.yaml
```

*N.B.: Ingress annotations are ingress controller dependent*