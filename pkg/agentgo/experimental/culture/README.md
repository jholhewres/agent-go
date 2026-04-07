# experimental/culture

> **Warning: Experimental — API may change without notice.**

In-memory store for cultural knowledge entries used to configure agent behaviour per organisation or locale.

## Purpose

Manages tagged `Entry` records (content + metadata) with CRUD operations, tag-based search, and UUID-keyed storage. Designed to supply contextual cultural norms or guidelines to agents at runtime.

## Main Types

- `Entry` — cultural knowledge record with ID, content, tags, metadata, timestamps
- `Manager` — thread-safe store: `Add`, `Get`, `Update`, `Delete`, `Search`, `List`

## Status

**experimental** — no consumers outside the package itself.
