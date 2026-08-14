# Minimal Agent Runtime

This runtime can answer directly or call three read-only tools: a safe arithmetic parser, local document search, and predefined PostgreSQL queries. Tool and document output is untrusted data. It cannot change runtime rules or authorize an operation.

The public trace reports only observed events: rounds, selected tools, redacted argument summaries, status, duration, safe result summaries, and the final stop reason. Hidden model reasoning and system prompts are never part of the trace.

