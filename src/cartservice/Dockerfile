# https://mcr.microsoft.com/v2/dotnet/sdk/tags/list
FROM --platform=linux/amd64 mcr.microsoft.com/dotnet/sdk:5.0.103 as builder
WORKDIR /app
COPY cartservice.csproj .
RUN dotnet restore cartservice.csproj -r linux-musl-x64
COPY . .
# Fix the issue on Debian 10: https://github.com/dotnet/dotnet-docker/issues/2470
ENV COMPlus_EnableDiagnostics=0
# Fix for Google.Protobuf.Tools detection in Docker
ENV PROTOBUF_TOOLS_OS=linux
ENV PROTOBUF_TOOLS_CPU=x64
RUN dotnet publish cartservice.csproj -p:PublishSingleFile=true -r linux-musl-x64 --self-contained true -p:PublishTrimmed=True -p:TrimMode=Link -c release -o /cartservice --no-restore

# https://mcr.microsoft.com/v2/dotnet/runtime-deps/tags/list
FROM --platform=linux/amd64 mcr.microsoft.com/dotnet/runtime-deps:5.0.3-alpine3.12-amd64
RUN GRPC_HEALTH_PROBE_VERSION=v0.3.6 && \
    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe
WORKDIR /app
COPY --from=builder /cartservice .
ENV ASPNETCORE_URLS http://0.0.0.0:7070
ENTRYPOINT ["/app/cartservice"]
