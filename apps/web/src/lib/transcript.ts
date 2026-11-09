import type { CaptionLine } from "~/lib/captions";

export type Paragraph = { at: string; lines: CaptionLine[] };

const paragraphSeconds = 60;

// Breaks the transcript into roughly minute-long paragraphs, each headed by the time
// its first line was said, so a long talk can be skimmed by timestamp.
export function groupTranscript(lines: CaptionLine[]): Paragraph[] {
	const paragraphs: Paragraph[] = [];
	for (const line of lines) {
		if (!line.final) continue;
		const current = paragraphs.at(-1);
		if (
			current &&
			Date.parse(line.at) - Date.parse(current.at) < paragraphSeconds * 1000
		)
			current.lines.push(line);
		else paragraphs.push({ at: line.at, lines: [line] });
	}
	return paragraphs;
}

export const lineAnchor = (lineId: string) => `line-${lineId}`;
