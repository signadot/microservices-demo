# Signadot sandboxes for this repo

Sandbox specs for iterating on the Online Boutique microservices against a
real cluster without rebuilding images per change. Each spec puts exactly one
service on your devbox while the rest of the stack stays in-cluster; traffic
with the sandbox routing key lands on your local process.

## Cluster / namespace

- **Cluster:** `demo`
- **Namespace:** `microservices-demo`

If those differ in your environment, edit the specs in `dev/` before applying.

## Workflow

1. Find your devbox ID:
   ```bash
   signadot devbox list   # or via MCP: list_devboxes
   ```
2. Apply the spec for the service you are changing:
   ```bash
   signadot sandbox apply -f .signadot/dev/checkoutservice.yaml \
     --set devbox-id=<devbox-id>
   ```
3. Wait for `ready: true` and tunnel `connected: true`:
   ```bash
   signadot sandbox get checkoutservice-dev
   ```
4. Run the local service from the devbox (see per-service recipes below).
5. Validate against `http://<svc>.microservices-demo.svc[:<port>]/...` with
   the `baggage: sd-routing-key=<key>` header.

## Per-service run recipes

All services read their downstream addresses from env vars. Resolve each
address to the FQDN that is in `/etc/hosts` on the devbox (`<svc>.microservices-demo`
or `<svc>.microservices-demo.svc`). Missing a downstream var does not fail at
startup — it fails on the first real request as a 500 or a timeout.

### checkoutservice (Go, port 5050)

```bash
cd src/checkoutservice
go build -o /tmp/checkoutservice .

cat > /tmp/start_checkout.sh << 'EOF'
#!/bin/bash
export PORT=5050
export PRODUCT_CATALOG_SERVICE_ADDR=productcatalogservice.microservices-demo.svc:3550
export SHIPPING_SERVICE_ADDR=shippingservice.microservices-demo.svc:50051
export PAYMENT_SERVICE_ADDR=paymentservice.microservices-demo.svc:50051
export EMAIL_SERVICE_ADDR=emailservice.microservices-demo.svc:5000
export CURRENCY_SERVICE_ADDR=currencyservice.microservices-demo.svc:7000
export CART_SERVICE_ADDR=cartservice.microservices-demo.svc:7070
exec /tmp/checkoutservice
EOF
chmod +x /tmp/start_checkout.sh

fuser -k 5050/tcp 2>/dev/null || true
setsid /tmp/start_checkout.sh >> /tmp/checkoutservice.log 2>&1 &
```

### frontend (Go, port 8080)

Note: the in-cluster Service exposes **port 80**, not 8080. Always curl
`http://frontend.microservices-demo.svc/` (implicit :80). See "Known gotchas".

```bash
cd src/frontend
go build -o /tmp/frontend .

cat > /tmp/start_frontend.sh << 'EOF'
#!/bin/bash
export PORT=8080
export PRODUCT_CATALOG_SERVICE_ADDR=productcatalogservice.microservices-demo.svc:3550
export CURRENCY_SERVICE_ADDR=currencyservice.microservices-demo.svc:7000
export CART_SERVICE_ADDR=cartservice.microservices-demo.svc:7070
export RECOMMENDATION_SERVICE_ADDR=recommendationservice.microservices-demo.svc:8080
export SHIPPING_SERVICE_ADDR=shippingservice.microservices-demo.svc:50051
export CHECKOUT_SERVICE_ADDR=checkoutservice.microservices-demo.svc:5050
export AD_SERVICE_ADDR=adservice.microservices-demo.svc:9555
exec /tmp/frontend
EOF
chmod +x /tmp/start_frontend.sh

fuser -k 8080/tcp 2>/dev/null || true
setsid /tmp/start_frontend.sh >> /tmp/frontend.log 2>&1 &
```

## Known gotchas

- **Frontend Service port is 80, not 8080.** The Dockerfile `EXPOSE 8080` is
  the container port; the Kubernetes Service in `kubernetes-manifests/frontend.yaml`
  maps `:80 → :8080`. Hit `http://frontend.microservices-demo.svc/` (no port).
  Using `:8080` on the `.svc` URL returns an Envoy 503 that looks like a pod
  health issue but is a port mismatch.

- **gRPC downstream addresses must be fully set.** Missing a `*_SERVICE_ADDR`
  env var does not fail at startup — it fails on the first request that needs
  that downstream. Grep the service's Go source for `*ServiceAddr` / `getEnv`
  before starting to enumerate every required var.

- **Baggage header propagation.** Services built from this repo use an
  instrumented gRPC client and forward `baggage` automatically. If you add a
  new service or a handler that uses raw `http.Client`, baggage will be
  dropped and sandboxed downstreams will not fire.

## Validation

From the devbox, drive the cluster `.svc` URL with the routing key header:

```bash
curl -s http://frontend.microservices-demo.svc/ \
  -H "baggage: sd-routing-key=<routing-key>"
```

Never validate against `http://localhost:<port>/` — local loopback bypasses
cluster routing, so downstream calls from your local process go to the
cluster baseline without the routing key and sandboxed downstream services
never fire.
