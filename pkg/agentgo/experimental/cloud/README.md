# experimental/cloud

> **Warning: Experimental — API may change without notice.**

Minimal deployment abstraction for publishing agent artifacts to cloud targets.

## Purpose

Defines a `Deployer` interface with a single `Deploy` method and a `NoopDeployer` placeholder for local/test usage. Intended as the foundation for future cloud-provider implementations (AWS Lambda, GCP Cloud Run, etc.).

## Main Types

- `Deployer` interface — `Deploy(ctx, artifact, config) (string, error)`
- `NoopDeployer` — returns `local://<artifact>`, useful for testing

## Status

**experimental** — no production implementations exist yet.
