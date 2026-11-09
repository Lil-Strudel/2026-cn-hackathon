import type { CaptionLine } from "~/lib/captions";
import { fetchTalk } from "~/lib/talks";
import { corrections, scriptFor, transcript } from "~/lib/transcript-fixtures";

const historyLines = 4;
const secondsBetweenLiveLines = 7;

export async function captionHistory(talkId: string): Promise<CaptionLine[]> {
	const talk = await fetchTalk(talkId);
	if (talk.status === "finished") return transcript(talkId, talk.startedAt);

	const now = Date.now();
	return scriptFor(talkId)
		.slice(0, historyLines)
		.map((line, index) => ({
			id: `${talkId}-${index}`,
			at: new Date(
				now - (historyLines - index) * secondsBetweenLiveLines * 1000,
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
