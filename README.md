# docker-swarm-autoscaler
A sidecar server that keeps your swarm scaled based on resource configuration



- [docker-swarm-autoscaler](#docker-swarm-autoscaler)
  - [Requirements](#requirements)
  - [Usage](#usage)
  - [Roadmap](#roadmap)



## Requirements
- Docker
- Docker Swarm (or docker.compose with `services` key)
- Labeled services (doesn't work for containers)



## Usage



## Roadmap

-[] Add embedded database for storing metrics between deployments
-[] Add support for multiple resource types (network, requests)
-[] Add support for prometheus metrics instead of docker stats
