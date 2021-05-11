# Not So Smart Cal

An experiment in microservices and kubernetes.

## Requirements

- [Docker](https://docs.docker.com/get-docker/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Minikube](https://minikube.sigs.k8s.io/docs/start/)

## Usage

Make a copy the env file:

`cp .env.example .env`

Start host services and kubernetes cluster:

`make dev`

The web frontend and GraphQL services are exposed at `http://localhost:3000/` and `http://localhost:4000/` respectively.
