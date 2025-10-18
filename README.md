# k8s-health-checker

Kubernetes cluster monitoring tool built with client-go informers.
Because my raspberry pi nodes have a death wish.

## What it does

Uses Kubernetes client-go library to register event handlers on informers. Watches pods, nodes, and deployments in real-time through informer event listeners (AddFunc, UpdateFunc, DeleteFunc).

When something goes wrong, you get notified via Discord or console output.

## Build

```bash
go build -o k3s-health-checker .
```

## Run

```bash
./k3s-health-checker -config config.yaml
```

Make sure you're running this inside the cluster or have kubeconfig set up properly.

## Configuration

Create a `config.yaml`:

```yaml
checker:
  check_pods: true
  check_nodes: true
  check_deployments: true

notifiers:
  discord:
    enabled: false
    webhook_url: "https://discord.com/api/webhooks/..."

  console:
    enabled: true
```

### Notifiers

Pick one:

- **Discord**: Set `enabled: true` and add your webhook URL
- **Console**: Just prints to stdout

If Discord is enabled, it takes priority. Otherwise falls back to console.

## Testing

```bash
go test ./...
```

## How it works

Built on top of Kubernetes client-go SharedInformer pattern:

1. Creates SharedInformerFactory from clientset
2. Registers event handlers (AddFunc, UpdateFunc, DeleteFunc) on pod/node/deployment informers
3. Informers maintain local cache and watch API server for changes
4. Event handlers check resource health status
5. Sends notifications when problems detected

No polling involved. Informers handle all the watch mechanisms and caching. Event handlers get called automatically when resources change state.

## Requirements

- Go 1.25+
- Access to Kubernetes cluster (runs in-cluster or with kubeconfig)
- Discord webhook URL (optional)

## License

See LICENSE file.
