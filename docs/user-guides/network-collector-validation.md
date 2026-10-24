# Validate the network collector in disposable Ubuntu

Status: the enabled container comparison **passed** on 2026-10-24. The
[implementation handoff](../specifications/prototype/network-collector.md#disposable-ubuntu-execution-verification)
records the image identity, tested kernel, exit status, and full log.
Ordinary `vorbere run test` remains synthetic and opens no test sockets.

Use this procedure only on an approved development machine with an existing
Docker daemon. It does not install Docker, modify services, change the host
firewall, or enable host networking. The Ubuntu container shares its Docker
host's Linux kernel. Record both the Ubuntu userland version and that kernel;
testing an Ubuntu-distributed kernel additionally requires an Ubuntu Docker host
or a disposable Ubuntu VM.

## What the validation does

The test downloads no runtime dependencies. Docker pulls the official
`ubuntu:24.04` image if needed. A locally compiled static test binary is mounted
read-only as the container's only test input. The container's root filesystem is
read-only, and output goes to the terminal. Docker retains the downloaded image
in its cache; `--rm` removes the test container after it exits.

The container has a dedicated PID namespace and a network namespace containing
only loopback. No ports are published to the host. The test opens eight temporary
IPv4/IPv6 TCP/UDP wildcard/loopback endpoints, establishes local TCP connections,
and creates connected and unbound UDP sockets. It sends no UDP datagram. TCP
traffic remains within the container's loopback interface. All sockets close on
exit, and no product service is contacted.

Docker's [none network driver](https://docs.docker.com/engine/network/drivers/none/)
may leave IPv6 loopback unconfigured. The command enables IPv6 only in the
container namespace, and grants `NET_ADMIN` solely to let the test add `::1/128`
to that namespace's loopback if missing. All other capabilities are dropped.
The test refuses socket creation unless it is root PID 1 on Ubuntu with exactly
one loopback interface and an explicit acknowledgement variable. The production
collector does not perform these test setup operations.

## Build and run after the environment is approved

From the repository root on a machine whose Docker daemon can see the temporary
directory, build the dedicated Linux/amd64 binary:

```sh
avera_live_dir=$(mktemp -d /tmp/avera-network-live.XXXXXX)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -tags networklive -c \
  -o "$avera_live_dir/network-live.test" ./internal/collect/network

docker pull --platform linux/amd64 ubuntu:24.04
docker image inspect --format '{{.Id}} {{json .RepoDigests}}' ubuntu:24.04

docker run --rm --platform linux/amd64 --pull never \
  --network none --read-only --user 0:0 \
  --cap-drop ALL --cap-add NET_ADMIN --security-opt no-new-privileges \
  --sysctl net.ipv6.conf.all.disable_ipv6=0 \
  --sysctl net.ipv6.conf.lo.disable_ipv6=0 \
  --pids-limit 64 --memory 256m --cpus 1 \
  --mount "type=bind,src=$avera_live_dir/network-live.test,dst=/network-live.test,readonly" \
  --env AVERA_NETWORK_LIVE=disposable-ubuntu \
  --entrypoint /network-live.test ubuntu:24.04 \
  -test.run '^TestLiveUbuntuEndpoints$' -test.v -test.timeout 45s
```

The [Docker run reference](https://docs.docker.com/engine/containers/run/)
documents image selection, mount options, namespace networking, and capabilities.
Keep the exact inspected image ID/digest with the result; the `24.04` tag can
change. Remove the generated temporary directory after retaining the log if
required. Do not remove unrelated Docker images or containers as cleanup.

## Interpreting the result

A successful run must execute `TestLiveUbuntuEndpoints` (not skip it), report
complete coverage, and match exactly eight expected endpoints and their own
executable/UID attribution. Extra established TCP, connected UDP or unbound UDP
findings fail the comparison. The log records OS metadata, kernel release,
architecture, and Go version. An unavailable IPv6 stack, denied namespace setup,
or incomplete process visibility fails visibly rather than claiming success.

Preserve the command's exit status, complete test log, image identity and Docker
host architecture with the [T02 handoff](../specifications/prototype/network-collector.md).
The test does not establish reachability, firewall behavior, arbitrary container
namespace traversal, every possible process race, or support for untested
kernels/architectures.
