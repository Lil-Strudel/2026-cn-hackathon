export type Correction = {
	/** Offsets into `text` of the corrected term. */
	start: number;
	end: number;
	/** What Whisper heard before the vocabulary corrected it. */
	heard: string;
};

export type CaptionLine = {
	id: string;
	at: string;
	text: string;
	/** Partial lines keep their id and are replaced until one arrives with `final: true`. */
	final: boolean;
	corrections: Correction[];
};

export type CaptionPart = { text: string; heard?: string; offset: number };

export function splitCorrections(line: CaptionLine): CaptionPart[] {
	const parts: CaptionPart[] = [];
	let offset = 0;
	const corrections = [...line.corrections].sort((a, b) => a.start - b.start);
	for (const { start, end, heard } of corrections) {
		if (start < offset || end > line.text.length) continue;
		if (start > offset)
			parts.push({ text: line.text.slice(offset, start), offset });
		parts.push({ text: line.text.slice(start, end), heard, offset: start });
		offset = end;
	}
	if (offset < line.text.length)
		parts.push({ text: line.text.slice(offset), offset });
	return parts;
}

export function mergeLines(
	lines: CaptionLine[],
	incoming: CaptionLine[],
): CaptionLine[] {
	const byId = new Map(lines.map((line) => [line.id, line]));
	for (const line of incoming) {
		// A late partial must not undo a line that has already been finalised.
		if (byId.get(line.id)?.final && !line.final) continue;
		byId.set(line.id, line);
	}
	return [...byId.values()].sort((a, b) => a.at.localeCompare(b.at));
}

const useFixtures = import.meta.env.VITE_API_FIXTURES !== "false";
const fixtures = () => import("~/lib/captions-fixtures");

export async function fetchCaptions(talkId: string): Promise<CaptionLine[]> {
	if (useFixtures) return (await fixtures()).captionHistory(talkId);
	const path = `/api/talks/${encodeURIComponent(talkId)}/captions`;
	const res = await fetch(path);
	if (!res.ok) throw new Error(`GET ${path}: ${res.status}`);
	return res.json();
}

export type StreamStatus = "connecting" | "open" | "reconnecting";

export function subscribeCaptions(
	talkId: string,
	onLine: (line: CaptionLine) => void,
	onStatus: (status: StreamStatus) => void,
): () => void {
	if (useFixtures) {
		let stop = () => {};
		let cancelled = false;
		fixtures().then(({ simulateCaptions }) => {
			if (cancelled) return;
			onStatus("open");
			stop = simulateCaptions(talkId, onLine);
		});
		return () => {
			cancelled = true;
			stop();
		};
	}

	const source = new EventSource(`/events/talks/${encodeURIComponent(talkId)}`);
	source.onopen = () => onStatus("open");
	// EventSource retries on its own; this only tells the reader it is doing so.
	source.onerror = () => onStatus("reconnecting");
	source.addEventListener("caption", (event) => onLine(JSON.parse(event.data)));
	return () => source.close();
}
