import {
	type Highlight,
	type HighlightField,
	type NewTalk,
	type SearchHit,
	type Talk,
	type TalkNotes,
	talkSlug,
} from "~/lib/talks";
import { transcript } from "~/lib/transcript-fixtures";

const today = (hours: number, minutes: number) => {
	const date = new Date();
	date.setHours(hours, minutes, 0, 0);
	return date.toISOString();
};

// Talks made with "New talk" are kept in the browser, so a demo survives a reload.
const createdKey = "created-talks";

function readCreated(): Talk[] {
	try {
		const stored = JSON.parse(localStorage.getItem(createdKey) ?? "[]");
		return Array.isArray(stored) ? stored : [];
	} catch {
		return [];
	}
}

const created = readCreated();

export function addTalk(input: NewTalk): Talk {
	const talk: Talk = {
		...input,
		id: talkSlug(input.title, new Set(talks.map((t) => t.id))),
		status: "live",
		startedAt: new Date().toISOString(),
		endedAt: null,
	};
	created.unshift(talk);
	talks.unshift(talk);
	try {
		localStorage.setItem(createdKey, JSON.stringify(created));
	} catch {}
	return talk;
}

export const talks: Talk[] = [
	...created,
	{
		id: "understanding-kubernetes",
		title: "Understanding Kubernetes",
		speaker: "Priya Natarajan",
		description:
			"A tour of the control plane: how the API server, etcd, the scheduler and controllers reconcile what you asked for with what is running.",
		status: "live",
		startedAt: today(14, 0),
		endedAt: null,
	},
	{
		id: "ebpf-for-the-rest-of-us",
		title: "eBPF for the rest of us",
		speaker: "Marcus Oyelaran",
		description:
			"What Cilium and Hubble do in the kernel, explained without assembly, and how to read a network flow when a pod can't reach its database.",
		status: "live",
		startedAt: today(14, 15),
		endedAt: null,
	},
	{
		id: "gitops-without-tears",
		title: "GitOps without tears",
		speaker: "Lena Hoffmann",
		description:
			"Flux, OCI artifacts and cosign: shipping signed manifests to a cluster that pulls its own updates.",
		status: "finished",
		startedAt: today(11, 0),
		endedAt: today(11, 40),
	},
	{
		id: "observability-from-day-one",
		title: "Observability from day one",
		speaker: "Tomás Reyes",
		description:
			"Instrumenting a new service with OpenTelemetry before the first bug report: traces, logs and metrics that answer real questions.",
		status: "finished",
		startedAt: today(10, 0),
		endedAt: today(10, 45),
	},
	{
		id: "autoscaling-on-events",
		title: "Autoscaling on events with KEDA",
		speaker: "Aiko Tanaka",
		description:
			"Scaling deployments on queue depth and connected clients instead of CPU, and the traps in scaling to zero.",
		status: "finished",
		startedAt: today(9, 0),
		endedAt: today(9, 30),
	},
	{
		id: "search-that-understands-you",
		title: "Search that understands you",
		speaker: "Daniel Okafor",
		description:
			"Hybrid search in Elasticsearch: when keyword matching wins, when vectors win, and how to combine both.",
		status: "finished",
		startedAt: today(13, 0),
		endedAt: today(13, 45),
	},
];

export function findTalk(id: string): Talk {
	const talk = talks.find((t) => t.id === id);
	if (!talk) throw new Error(`GET /api/talks/${id}: 404`);
	return talk;
}

const notes: Record<string, TalkNotes> = {
	"gitops-without-tears": {
		summary:
			"Lena argued that clusters should pull their state rather than have it pushed. Flux applies signed OCI artifacts built by CI, verifying each cosign signature first, so the cluster runs exactly what was reviewed and a rollback is a Git revert. The closing section covered keeping secrets out of the repository.",
		notes: [
			"The cluster pulls from Git; nobody pushes into it.",
			"Flux can watch an OCI artifact instead of a branch, so it applies exactly what CI built.",
			"Unsigned or tampered artifacts are ignored after the cosign check.",
			"Rollback is a revert; cluster history is repository history.",
		],
	},
	"observability-from-day-one": {
		summary:
			"Tomás walked through instrumenting a new service with OpenTelemetry before it has users. Traces come first because they show where time goes, logs carry the trace ID so they link back, and metrics are kept to the few questions asked every day.",
		notes: [
			"Instrument before the first outage, not after.",
			"Traces first, then logs with trace IDs, then a small set of metrics.",
			"Delete dashboards nobody opens.",
		],
	},
	"autoscaling-on-events": {
		summary:
			"Aiko showed why CPU is a poor scaling signal for queue workers and how KEDA scales on queue depth or connected clients by feeding external metrics to the HPA. Scaling to zero saves money but brings cold starts, and a cooldown stops the deployment from flapping.",
		notes: [
			"Scale on the work waiting, not on CPU.",
			"KEDA feeds external metrics to the horizontal pod autoscaler.",
			"Set a cooldown to avoid flapping; plan for cold starts at zero.",
		],
	},
	"search-that-understands-you": {
		summary:
			"Daniel compared keyword and vector search in Elasticsearch and showed how reciprocal rank fusion combines them without hand-tuned weights. Phonetic analysis catches misheard and misspelled names, and relevance should be measured on real user queries.",
		notes: [
			"BM25 for exact terms, dense vectors for meaning; use both.",
			"Reciprocal rank fusion merges the two lists without tuning weights.",
			"Phonetic analysis helps with names speech recognition gets wrong.",
		],
	},
};

const pendingNotes: TalkNotes = { summary: null, notes: null };

export function notesFor(id: string): TalkNotes {
	return notes[findTalk(id).id] ?? pendingNotes;
}

const escapeRegExp = (text: string) =>
	text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

const fragmentRadius = 60;

function highlight(
	field: HighlightField,
	text: string,
	pattern: RegExp,
): Highlight | null {
	const first = text.search(pattern);
	if (first === -1) return null;
	const start =
		first <= fragmentRadius ? 0 : text.indexOf(" ", first - fragmentRadius) + 1;
	const cut = text.lastIndexOf(" ", first + fragmentRadius * 2);
	const end =
		first + fragmentRadius * 2 >= text.length || cut <= first
			? text.length
			: cut;
	const excerpt = `${start > 0 ? "…" : ""}${text.slice(start, end)}${end < text.length ? "…" : ""}`;
	return { field, fragment: excerpt.replace(pattern, "<mark>$&</mark>") };
}

const transcriptHitsPerTalk = 3;

export function search(query: string): SearchHit[] {
	const pattern = new RegExp(escapeRegExp(query.trim()), "gi");
	return talks.flatMap((talk) => {
		const { summary, notes } = notesFor(talk.id);
		const transcriptHits =
			talk.status !== "finished"
				? []
				: transcript(talk.id, talk.startedAt)
						.map((line) => {
							const hit = highlight("transcript", line.text, pattern);
							return hit && { ...hit, lineId: line.id, at: line.at };
						})
						.filter((h) => h !== null)
						.slice(0, transcriptHitsPerTalk);
		const highlights = [
			highlight("title", talk.title, pattern),
			highlight("description", talk.description, pattern),
			summary === null ? null : highlight("summary", summary, pattern),
			...(notes ?? []).map((note) => highlight("notes", note, pattern)),
			...transcriptHits,
		].filter((h) => h !== null);
		return highlights.length > 0 ? [{ talk, highlights }] : [];
	});
}
