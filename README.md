# Cloud Native Hackathon 2026

Real-time captions for conference speakers, built at the
[WeMakeDevs Cloud Native Hackathon](https://www.wemakedevs.org/hackathons/cnh) on November 9, 2026.
The product name is chosen by the team on the day.

## Architecture

```
browser (React SPA, Dash0 RUM)
   │  Cloudflare Tunnel (outbound-only from the cluster)
   ├── /api/*     → api     (Go)  Whisper transcription · Elasticsearch · publishes captions to Redis
   ├── /events/*  → fanout  (Go)  Redis pub/sub → Server-Sent Events, autoscaled by KEDA on connected clients
   └── /*         → web     (Caddy serving the Vite build)

Redis (pub/sub only, never scaled) · Elasticsearch (ECK, analysis-phonetic plugin)
OpenTelemetry everywhere → Dash0
```

Elasticsearch is the core data layer:

- **Phonetic correction.** Whisper output is matched against a vocabulary with phonetic
  analysis, so a misheard term snaps to the right spelling. A term that sounds like an everyday
  word carries exception words and isn't used when one is said nearby ("Sonar Gitar", but
  "playing a guitar").
- **Vocabulary prompting.** Significant-terms aggregations over a talk's description seed
  Whisper's prompt with the jargon it should expect.
- **Transcript search.** Search across every talk's captions, jumping to the moment a term was said.
- **Event discovery.** Find talks you may like from what you have watched.

## Built with

|                 |                                                                                         |
| --------------- | --------------------------------------------------------------------------------------- |
| Search and data | Elasticsearch (ECK), with the `analysis-phonetic` plugin                                |
| Transcription   | OpenAI Whisper                                                                          |
| Observability   | OpenTelemetry, Dash0 (traces, logs, metrics, RUM, Agent0)                               |
| Kubernetes      | Talos homelab cluster, Cilium + Hubble, KEDA, Flux (OCI artifacts verified with cosign) |
| Cluster UI      | Radar                                                                                   |
| Code review     | Gitar                                                                                   |
| Edge            | Cloudflare Tunnel                                                                       |
| App             | Go, React, TanStack Router and Query, Tailwind, shadcn/ui, Vite, Biome, pnpm, dekit     |

## Development

Prerequisites: [mise](https://mise.jdx.dev) and Docker.

```bash
mise install          # go, node, pnpm, golangci-lint, air, kustomize, kubeconform, radar
pnpm install
cp .env.example .env  # optional: OpenAI key, Dash0 endpoint
pnpm dev              # Elasticsearch, Redis, api, fanout and web under dekit
```

Until the api serves talks, the web app reads sample talks from
`apps/web/src/lib/*-fixtures.ts`. Set `VITE_API_FIXTURES=false` in `.env` to call the real
`/api/talks` endpoints instead. Captions are always real unless you set
`VITE_CAPTION_FIXTURES=true`, which plays simulated captions for a demo without a speaker.

To caption a talk, open a live one such as http://localhost:4000/talks/understanding-kubernetes,
choose "Caption this talk" and allow the microphone. Everyone on the talk page sees the captions
as you speak, and people who join later get the earlier lines too. "New talk" on the home page
starts a live talk of your own and opens it for captioning; with fixtures it is kept in the
browser's `localStorage`.

The app runs at http://localhost:4000. Vite proxies `/api` to the api on `:4100` and `/events`
to fanout on `:4200`. `pnpm dev:down` stops everything. Kibana is available as the
`infra/kibana` task in the dekit UI.

On "Try correction", "Speak" transcribes your microphone without captioning any talk, and
corrects what was heard when you stop.

Without `OPENAI_API_KEY`, the api transcribes with a fake that turns every two seconds of audio
into a canned line, misheard the way Whisper would, so the whole caption path and its
corrections run without a key. Turn off "Correct terms" above a talk's captions to see them
as Whisper heard them.

| Command                               |                                                         |
| ------------------------------------- | ------------------------------------------------------- |
| `pnpm check`                          | Biome lint and format check (`pnpm check:fix` to apply) |
| `pnpm typecheck`                      | TypeScript                                              |
| `pnpm test`                           | Vitest                                                  |
| `go test ./...` / `golangci-lint run` | Go tests and lint                                       |

## Delivery

Every change lands through a pull request: CI must pass, the title must be a
[Conventional Commit](https://www.conventionalcommits.org), and Gitar reviews it. PRs are
squash-merged.

On merge to `main`, CI does the following:

1. Builds the four images (`api`, `fanout`, `web`, `elasticsearch`) with SBOM and provenance attestations.
2. Signs them with keyless cosign.
3. Pins `deploy/k8s` to the commit's images and pushes it to GHCR as a signed OCI artifact.

Flux in the cluster verifies the signature and applies the manifests, then posts the result
back to the commit as a status.

## Prepared before the event

The rules allow templates and boilerplate. Everything merged before the event was prepared on
November 8, in pull requests #1 to #4:

- the cluster operators
- the deploy pipeline
- this repository's tooling
- services that start, report health and emit telemetry, without product features
- instructions for coding agents (`AGENTS.md`, a UI components skill, a placeholder style guide)

All product work starts at the hackathon.

## AI disclosure

This project was built with the help of AI coding assistants (Claude Code).
