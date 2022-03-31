Build with :

CGO_ENABLED=0 go build -o ./build/kind -a -ldflags '-extldflags "-static"' .

Then create config file like

kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
networking:
  apiServerAddress: "10.166.240.217"
  apiServerPort: 6443


and then run 


kind create cluster --config config.yaml
