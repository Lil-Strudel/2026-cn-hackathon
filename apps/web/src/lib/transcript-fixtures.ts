import type { CaptionLine, Correction } from "~/lib/captions";

export type ScriptLine = { text: string; corrected?: Record<string, string> };

const kubernetes: ScriptLine[] = [
	{
		text: "Thanks for coming, everyone. Let's start with what actually happens when you run a deployment.",
	},
	{
		text: "You type kubectl apply, and the first thing that hears about it is the API server.",
		corrected: { kubectl: "cube control" },
	},
	{
		text: "The API server validates the object and writes it to etcd. That is the only place cluster state lives.",
		corrected: { etcd: "at CD" },
	},
	{
		text: "Nothing has started running yet. All we have is a record of what you asked for.",
	},
	{
		text: "Now the controllers wake up. Each one watches a kind of object and compares desired state with actual state.",
	},
	{
		text: "The ReplicaSet controller sees three replicas wanted and zero running, so it creates three pods.",
		corrected: { ReplicaSet: "replica set" },
	},
	{
		text: "Those pods have no node yet. That is the scheduler's job: it filters nodes that fit, then scores them.",
	},
	{
		text: "Once a pod is bound to a node, the kubelet on that node pulls the image and starts the containers.",
		corrected: { kubelet: "cubelet" },
	},
	{
		text: "And if a node dies, nobody runs a script. The controller simply notices the gap and closes it again.",
	},
	{
		text: "That loop, observe, compare, act, is the whole idea. Everything else is detail.",
	},
];

const ebpf: ScriptLine[] = [
	{
		text: "Let's talk about what happens to a packet before your application ever sees it.",
	},
	{
		text: "eBPF lets us run small, verified programs inside the kernel, right where the packet arrives.",
		corrected: { eBPF: "EBP F" },
	},
	{
		text: "Cilium uses that to replace iptables, so policy lookups stay fast no matter how many services you have.",
		corrected: { Cilium: "Sillium", iptables: "IP tables" },
	},
	{
		text: "And because the kernel already sees every flow, Hubble can show them to you without a sidecar.",
	},
	{
		text: "So when a pod can't reach its database, you can see exactly which policy dropped the connection.",
	},
];

const gitops: ScriptLine[] = [
	{
		text: "Raise your hand if you have ever run kubectl apply from a laptop on a Friday afternoon.",
	},
	{
		text: "GitOps flips that around. The cluster pulls what is in Git, and nobody pushes into the cluster at all.",
		corrected: { GitOps: "get ops" },
	},
	{
		text: "Flux watches an OCI artifact instead of a branch, so what it applies is exactly what CI built.",
	},
	{
		text: "Before it applies anything, Flux checks the cosign signature, so a tampered artifact is simply ignored.",
		corrected: { cosign: "co-sign" },
	},
	{
		text: "Rollback is a revert. The history of your cluster is the history of your repository.",
	},
	{
		text: "The hard part is secrets, and we will spend the last ten minutes on exactly that.",
	},
];

const observability: ScriptLine[] = [
	{
		text: "Most teams add observability after the first outage. Let's do it before the first bug report.",
	},
	{
		text: "OpenTelemetry gives us one API for traces, logs and metrics, and one wire format to ship them.",
		corrected: { OpenTelemetry: "open telemetry" },
	},
	{
		text: "Start with traces. A single trace answers where the time went for one request.",
	},
	{
		text: "Then put the trace ID on every log line, so a log takes you straight to its trace.",
	},
	{
		text: "Metrics come last, and only for the questions you ask every day, like error rate and latency.",
	},
	{ text: "If a dashboard nobody opens costs you money, delete it." },
];

const keda: ScriptLine[] = [
	{
		text: "CPU is a terrible signal for a service that mostly waits on a queue.",
	},
	{
		text: "KEDA scales a deployment on the thing that actually matters, like queue depth or connected clients.",
		corrected: { KEDA: "Keda" },
	},
	{
		text: "Under the hood it feeds an external metric to the horizontal pod autoscaler.",
	},
	{
		text: "Scaling to zero is the fun part, and also where the cold start surprises live.",
	},
	{
		text: "Set a sensible cooldown, or you will scale up and down every thirty seconds.",
	},
];

const search: ScriptLine[] = [
	{
		text: "Keyword search finds the exact term. Vector search finds the meaning. You usually want both.",
	},
	{
		text: "In Elasticsearch, BM25 handles keywords and a dense vector field handles semantics.",
		corrected: { Elasticsearch: "elastic search", BM25: "BM 25" },
	},
	{
		text: "Reciprocal rank fusion merges the two result lists without having to tune weights by hand.",
	},
	{
		text: "Phonetic analysis helps with names people spell differently, or that speech recognition mishears.",
	},
	{
		text: "Measure with real queries from your users, not the ones you think they will type.",
	},
];

export const scripts: Record<string, ScriptLine[]> = {
	"understanding-kubernetes": kubernetes,
	"ebpf-for-the-rest-of-us": ebpf,
	"gitops-without-tears": gitops,
	"observability-from-day-one": observability,
	"autoscaling-on-events": keda,
	"search-that-understands-you": search,
};

export const scriptFor = (talkId: string) => scripts[talkId] ?? kubernetes;

export function corrections(line: ScriptLine): Correction[] {
	return Object.entries(line.corrected ?? {}).map(([term, heard]) => {
		const start = line.text.indexOf(term);
		return { start, end: start + term.length, heard };
	});
}

const secondsBetweenLines = 25;

export function transcript(talkId: string, startedAt: string): CaptionLine[] {
	const start = Date.parse(startedAt);
	return scriptFor(talkId).map((line, index) => ({
		id: `${talkId}-${index}`,
		at: new Date(start + index * secondsBetweenLines * 1000).toISOString(),
		text: line.text,
		final: true,
		corrections: corrections(line),
	}));
}
