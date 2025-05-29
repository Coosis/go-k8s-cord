A poor attempt to do coordination between k8s clusters(In Progress). 

# Setup
Central requires etcd running at `localhost:2379`. See more details after config files are generated.
## With justfile
```bash
just gen-ca
just gen-central
just gen-agent
just gen-grpc

go run ./cmd/central serve # start central controller
go run ./cmd/agent serve # start agent controller
```
## Without justfile
Well, you can just copy the commands from the justfile and run them manually.

# Code Overview
Uses grpc to communicate between central and remote agents. 
Central have a single grpc server and multiple grpc clients, 
one for each remote agent.
Agents each have a grpc server and a grpc client to communicate with the central agent.

Agent sends heartbeats regularly to the central agent to prove 
liveness. 

# Interaction
## Central
```bash
go run ./cmd/central status
```

## Agent
```bash
go run ./cmd/agent status
go run ./cmd/agent pods
```
