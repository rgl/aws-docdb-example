# About

[![Build](https://github.com/rgl/aws-docdb-example/actions/workflows/build.yml/badge.svg)](https://github.com/rgl/aws-docdb-example/actions/workflows/build.yml)

AWS DocumentDB example Go application.

This is used in:

* [rgl/terramate-aws-eks-example](https://github.com/rgl/terramate-aws-eks-example)

# Usage

Install Docker Compose.

Create the environment:

```bash
docker compose up --build --detach
```

Access the example application endpoint:

```bash
xdg-open http://localhost:8000
```

Destroy the environment:

```bash
docker compose down --remove-orphans --volumes --timeout=0
```
