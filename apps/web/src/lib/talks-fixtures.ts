import type { Highlight, HighlightField, SearchHit, Talk } from "~/lib/talks";

const today = (hours: number, minutes: number) => {
	const date = new Date();
	date.setHours(hours, minutes, 0, 0);
	return date.toISOString();
};

export const talks: Talk[] = [
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
	const start = Math.max(0, first - fragmentRadius);
	const end = Math.min(text.length, first + fragmentRadius * 2);
	const excerpt = `${start > 0 ? "…" : ""}${text.slice(start, end)}${end < text.length ? "…" : ""}`;
	return { field, fragment: excerpt.replace(pattern, "<mark>$&</mark>") };
}

export function search(query: string): SearchHit[] {
	const pattern = new RegExp(escapeRegExp(query.trim()), "gi");
	return talks.flatMap((talk) => {
		const highlights = [
			highlight("title", talk.title, pattern),
			highlight("description", talk.description, pattern),
		].filter((h) => h !== null);
		return highlights.length > 0 ? [{ talk, highlights }] : [];
	});
}
