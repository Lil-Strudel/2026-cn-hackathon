# Known gaps

Found in the readiness check on November 8, the day before the event. Remove each item when it
is fixed.

## Product

- **KEDA autoscaling for fanout is not built.** The README says fanout is autoscaled by KEDA on
  connected clients, but `deploy/k8s` has no `ScaledObject`, and fanout exposes no client count
  yet. The homelab repo expects this repo to own the `ScaledObject`.
- **The KEDA network policy targets the wrong service.** The `keda` `CiliumNetworkPolicy` in the
  homelab repo (`kubernetes/hackathon/configs/cnh/network-policy.yaml`) lets KEDA reach `api` on
  port 9090. Fanout is the service that scales, so it probably needs to target `fanout` on the
  port fanout serves its metric on. Fix it in the homelab repo together with the
  `ScaledObject`.
- **fanout records no Redis metrics.** It calls `redisotel.InstrumentTracing` but not
  `InstrumentMetrics`, unlike the api. Depth of instrumentation counts for the Dash0 track.

## Repository

- **Renovate runs before 6am on Monday, the morning of the event.** Its dependency PRs would
  arrive during the event. Pause it (`.github/renovate.json`) until after the event.
- **Gitar's review is not a required check.** The `main` ruleset requires a PR and the `ci-ok`
  check, with no approvals, so a PR can merge before Gitar has reviewed it. The Gitar track
  scores features landing through PRs that Gitar reviewed. Add the `Gitar` check to the
  ruleset's required status checks.
- **No Gitar review instructions or rules.** Gitar already reads `AGENTS.md` and
  `.agents/skills/`. The Gitar track also scores project-aware review instructions
  (`.gitar/review/*.md`), repository rules (`.gitar/rules/*.md`) and auto-apply, which is turned
  on per PR with a `gitar auto-apply:on` comment or the `gitar-managed` label. CI failure
  diagnosis is on by default for GitHub Actions.
- **No license.** The rules don't require one, but without a license nobody else may reuse the
  public code.
