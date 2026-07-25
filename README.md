[![Workforce CI Pipeline](https://github.com/esdatalabs/workforce/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/esdatalabs/workforce/actions/workflows/ci.yml) [![CodeQL Advanced](https://github.com/esdatalabs/workforce/actions/workflows/codeql.yml/badge.svg)](https://github.com/esdatalabs/workforce/actions/workflows/codeql.yml) [![Dependabot Updates](https://github.com/esdatalabs/workforce/actions/workflows/dependabot/dependabot-updates/badge.svg)](https://github.com/esdatalabs/workforce/actions/workflows/dependabot/dependabot-updates)

# Workforce
Barebones implementation of the worker pool (Fan Out/In) pattern in go

## Fan Out/Fan In

```mermaid
flowchart LR
    D(job_n . . . job_1, job_0 )
    D --> B["worker_0[job_0]"]
    D --> C["worker_1[job_1]"]
    D --> E["worker_n[job_n]"]
    F(result_n . . .result_1, result_0)
    B --> F
    C --> F
    E --> F
```

## Installation

```sh
go install github.com/esdatalabs/workforce@latest
```

## Usage Example

[Batch Processing](./examples/batches/main.go)
