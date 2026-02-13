# dns_exporter
Prometheus DNS Metrics Exporter

# Clone
```bash
git clone git@github.com/crooks/dns_exporter
cd dns_exporter
```

# Build
## Docker/podman

```bash
docker build --tag dns_exporter:latest --file Dockerfile .
```

```bash
podman build --tag dns_exporter:latest --file Dockerfile .
```

## Go
```bash
go build
```

# Run
## Docker/podman

### Local build
```bash
docker run --detach --name dns_exporter --publish 0.0.0.0:9117:9117 localhost/dns_exporter:latest
```

```bash
podman run --detach --name dns_exporter --publish 0.0.0.0:9117:9117 localhost/dns_exporter:latest
```

### GitHub Container Registry image
```bash
docker run --detach --name dns_exporter --publish 0.0.0.0:9117:9117 ghcr.io/crooks/dns_exporter:main
```

```bash
podman run --detach --name dns_exporter --publish 0.0.0.0:9117:9117 ghcr.io/dns_exporter:main
```

## Linux systemd

As root or prefixing all commands with `sudo`:

1. Copy the binary from the build or extract it from the container image to `/usr/local/bin/dns_exporter`
1. Add a new user to run the exporter: `useradd --shell /bin/false --user-group --no-create-home --comment "dns_exporter user"`
1. Create with appropriate permissions `install --mode 700 --directory --owner dns_exporter --group dns_exporter /etc/dns_exporter/`
1. Copy and update the `examples/config.yml` file to `/etc/dns_exporter`: `install --mode 600 --owner dns_exporter --group dns_exporter examples/config.yml /etc/dns_exporter`
1. Create `/etc/systemd/system/dns_exporter.service` with contents:
```ini
[Unit]
Description=DNS Exporter

[Service]
User=dns_exporter
Group=dns_exporter
ExecStart=/usr/local/bin/dns_exporter
Restart=on-failure

# Harden the service
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=full
ProtectHome=yes
ReadOnlyPaths=/etc/dns_exporter/
InaccessiblePaths=/root /proc/kcore /sys/firmware
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE
RestrictAddressFamilies=AF_INET AF_INET6
SystemCallFilter=@system-service
SystemCallFilter=~@privileged @resources
UMask=0077
MountFlags=private
PrivateIPC=yes
CPUQuota=10%
MemoryHigh=64M
MemoryMax=128M
TasksMax=100
StandardOutput=journal

[Install]
WantedBy=multi-user.target
```
1. Read the new service: `systemctl daemon-reload`
1. Start and enable the service: `systemctl enable --now dns_exporter.service`

You will need to open up port 9117/TCP on your firewall to allow your prometheus server to scrape the metrics. That is left as an exercise for the reader.

## Kubernetes
1. Create a deployment:
```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dnsexporter
spec:
  replicas: 1
  selector:
    matchLabels:
      app: dnsexporter
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
        -
          image: ghcr.io/crooks/dns_exporter:main
          imagePullPolicy: Always
          livenessProbe:
            exec:
              command:
                - curl
                - -q
                - localhost:9117/metrics
            initialDelaySeconds: 5
            periodSeconds: 10
          name: dnsexporter
          ports:
            - containerPort: 9117
          readinessProbe:
            periodSeconds: 5
            tcpSocket:
              port: 9117
          resources:
            limits:
              cpu: 250m
              memory: 256Mi
            requests:
              cpu: 100m
              memory: 128Mi
          volumeMounts:
            - mountPath: /app/config.yml
              name: config
              subPath: config.yml
              readOnly: true
      restartPolicy: Always
      volumes:
        - configMap:
            name: config
          name: config
```
1. Create a service:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: dnsexporter
spec:
  ports:
    - port: 9117
  selector:
    app: dnsexporter
```
1. Create an ingress:
```yaml
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: dnsexporter
spec:
  rules:
    - host: dnsexporter.example.com
      http:
        paths:
          - backend:
              service:
                name: dnsexporter
                port:
                  number: 80
            path: /
            pathType: Prefix
```
1. Create a configmap for the `config.yml` file:
```yaml
apiVersion: v1
data:
  config.yml: |
    ---
    logging:
      level: debug
    default_ns: 8.8.4.4
    resolve:
      eu-west-2.console.aws.amazon.com:
        nameservers:
          - 8.8.8.8:53
      google.co.uk:
        nameservers:
          - 81.187.219.204:53
          - 81.187.219.202:53
      fleegle.mixmin.net:
      unregistered.nonesense:
      google.com:
        nameservers:
          - 11.12.13.14
kind: ConfigMap
metadata:
  creationTimestamp: null
  name: config

```

