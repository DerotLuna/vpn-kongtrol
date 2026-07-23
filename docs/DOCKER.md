# Docker

For headless / server deployments. The container requires elevated network privileges to manage kernel tunnel interfaces.

```bash
docker build -f build/docker/Dockerfile -t kongtrol .

docker run -d \
  --name kongtrol \
  --privileged \
  --cap-add NET_ADMIN \
  --device /dev/net/tun \
  --network host \
  -v ~/.kongtrol:/etc/kongtrol:ro \
  -p 127.0.0.1:9741:9741 \
  kongtrol
```

Or with Compose:

```bash
docker compose -f build/docker/docker-compose.yml up -d
```

> **Note:** `--privileged` and `--cap-add NET_ADMIN` are required for tunnel and routing management. The control API only accepts loopback binds; expose it through a local SSH tunnel rather than publishing it on a LAN interface.
