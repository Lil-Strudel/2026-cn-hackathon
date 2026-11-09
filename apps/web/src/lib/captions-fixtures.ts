import type { CaptionLine, Correction } from "~/lib/captions";

type ScriptLine = { text: string; corrected?: Record<string, string> };

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

const scripts: Record<string, ScriptLine[]> = {
	"understanding-kubernetes": kubernetes,
	"ebpf-for-the-rest-of-us": ebpf,
};

const scriptFor = (talkId: string) => scripts[talkId] ?? kubernetes;

function corrections(line: ScriptLine): Correction[] {
	return Object.entries(line.corrected ?? {}).map(([term, heard]) => {
		const start = line.text.indexOf(term);
		return { start, end: start + term.length, heard };
	});
}

const historyLines = 4;
const secondsBetweenLines = 7;

export function captionHistory(talkId: string): CaptionLine[] {
	const now = Date.now();
	return scriptFor(talkId)
		.slice(0, historyLines)
		.map((line, index) => ({
			id: `${talkId}-${index}`,
			at: new Date(
				now - (historyLines - index) * secondsBetweenLines * 1000,
			).toISOString(),
			text: line.text,
			final: true,
			corrections: corrections(line),
		}));
}

const msPerWord = 320;

export function simulateCaptions(
	talkId: string,
	onLine: (line: CaptionLine) => void,
): () => void {
	const script = scriptFor(talkId);
	let index = historyLines;
	let words = 0;
	let at = new Date().toISOString();

	const timer = setInterval(() => {
		const line = script[index % script.length];
		if (!line) return;
		const all = line.text.split(" ");
		words += 1;
		const final = words >= all.length;
		onLine({
			id: `${talkId}-${index}`,
			at,
			text: all.slice(0, words).join(" "),
			final,
			corrections: final ? corrections(line) : [],
		});
		if (final) {
			index += 1;
			words = 0;
			at = new Date().toISOString();
		}
	}, msPerWord);

	return () => clearInterval(timer);
}
