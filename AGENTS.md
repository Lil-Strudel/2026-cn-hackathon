# AGENTS.md

Read `README.md` first. It covers the architecture, the dev setup, the commands and the delivery
pipeline. This file adds the hackathon rules and the conventions the README doesn't state.

## Hackathon rules

This repo is an entry in the [WeMakeDevs Cloud Native Hackathon](https://www.wemakedevs.org/hackathons/cnh/rules)
on Monday, November 9, 2026, in Salt Lake City. Submissions close at **5:30 PM Mountain Time**, and
presentations run from 5:30 to 6:30 PM. Breaking a rule can disqualify the team.

Each rule, followed by what it means here:

1. **Main coding and design start at the event. Ideas, notes and research beforehand are fine.**
   The boilerplate listed in the README under "Prepared before the event" is the only work done
   in advance. Don't backdate work or bring in product code written earlier.
2. **Frameworks, libraries, public APIs, templates, third-party tools and public assets are
   allowed. Judges score the original work done on the day.** Use libraries freely. The product
   logic must be written here.
3. **Submissions need a public source repository and a README with setup instructions.** The
   repo is public. If you change how the project is set up or run, update the README's
   Development section in the same PR.
4. **Use only data, tools and accounts you own or are allowed to use. Keep private, personal and
   login-protected information out of the repo and the demo.** Never commit secrets, API keys,
   tokens or `.env`; put new variables in `.env.example` with empty values. Don't commit
   recordings, transcripts or personal data from people who haven't agreed.
5. **AI coding assistants are allowed, but their use must be disclosed.** The README's "AI
   disclosure" section covers this for the whole project. It is not a reason to attribute
   individual commits or PRs (see [Commits](#commits)).
6. **You must understand the code you submit and be able to explain the architecture and the
   decisions behind it.** Keep PRs small enough to review. Put the reasoning behind non-obvious
   choices in the PR description so the team can explain them to judges.
7. **Projects generated entirely by AI, without meaningful contribution, verification or
   understanding, may be rejected.** Don't produce large unreviewed changes. Verify that things
   work (tests, running the app) and say what you checked.
8. **One project per team. Duplicate, copied or previously finished projects are
   disqualified.** Don't copy another project's product code.
9. **Treat participants, organisers, sponsors, speakers and judges with respect. Plagiarism or
   attempts to manipulate judging disqualify.**

Final eligibility and judging decisions rest with WeMakeDevs.

### Tracks

A project enters every sponsor track whose tool it genuinely uses, plus Merge Forward. Every
project is also in the running for best overall, and every track rewards making conferences
easier. What the judges look for, and how it affects the work:

- **Gitar (Sonar): "Never Merge Broken".** Features land through PRs that Gitar reviewed, one per
  feature, not a single PR at the end. Review instructions and repository rules count, as does
  using CI failure diagnosis and auto-apply.
- **Dash0: "Stop Flying Blind".** Depth of instrumentation: traces, logs, metrics and RUM in the
  frontend. Observability should be part of the product. Instrument new code with OpenTelemetry
  as you write it (`internal/telemetry`, `apps/web/src/lib/telemetry.ts`). Using Agent0 or the
  Dash0 MCP to find and fix a real problem during the day goes in the demo; a dashboard or SLO is a bonus.
- **Radar (Skyhook): "AI SRE".** The project runs on Kubernetes with Radar connected. Use Radar's
  UI and MCP while building and operating, and its AI investigations on a real issue.
- **Elasticsearch: "Find the Right Talk".** Elasticsearch is the core data layer, not a side
  feature. Make good use of keyword, semantic, vector or hybrid search.
- **Merge Forward.** The project that makes conferences most inclusive and accessible. Live
  captions sit squarely here, so treat accessibility (contrast, keyboard use, screen readers,
  caption readability) as a feature.

## Pull requests

Everything lands through a pull request. Never commit or push to `main`.

- Branch from `main`, one branch per feature or fix, and open the PR as soon as the change works.
  Several small PRs beat one big one, for review and for the Gitar track.
- Fill in `.github/pull_request_template.md`. "How to verify" lists what you actually ran.
- Keep PR bodies short. Use a few bullets per section, one line each, and leave out anything the diff
  already shows. "Why" explains only the decisions a reviewer couldn't guess.
- PRs are squash-merged, so the PR title becomes the commit on `main`.

## Commits

Before every commit, stage your changes and run these checks, then fix every failure and stage
the fixes. There is no pre-commit hook; this is your job. Don't commit with a known failure, and
don't silence one with an ignore comment or by turning off a rule.

1. `pnpm exec biome check --write --staged` formats, lints and sorts imports in the staged
   files. Fix by hand whatever it can't fix itself.
2. If Go changed: `golangci-lint fmt` (gofumpt, goimports), then `golangci-lint run ./...`.
3. `pnpm typecheck` and `pnpm test`, plus `go test ./...` if Go changed.

- Every commit message and PR title is a [Conventional Commit](https://www.conventionalcommits.org):
  `type(scope): summary`, for example `feat(api): correct captions against the vocabulary`. CI
  rejects PR titles that aren't.
- **No attribution of any kind.** No `Co-Authored-By` trailers, no "Generated with Claude Code"
  or similar lines, in commits, PR titles, PR descriptions or code. AI use is disclosed once,
  in the README.

## Comments in code

We follow Rob Pike ("Notes on Programming in C", Comments) and Jeff Atwood ("Coding Without Comments").

- Clear code with good names comes first. If code needs a comment to be understood, try
  rewriting it before commenting it.
- Never write comments that restate what the code does. Comment only the non-obvious _why_: a
  constraint, a workaround, a surprising decision.
- Comments go stale and mislead; fewer, accurate comments beat many.
- No comment litter, section banners or commented-out code.
- Config files (YAML, TOML, JSON, Kubernetes manifests, etc.) get no comments.

## UI

For any UI work in `apps/web`, use the `ui-components` skill (`.agents/skills/ui-components/SKILL.md`).
It covers when to make a component, copying it from the shadcn/ui registry, building one on a
Radix primitive when shadcn has none, and styling it from `docs/style-guide.md`.
