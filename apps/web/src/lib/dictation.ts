import type { Transcript } from "~/lib/microphone-stream";

/** Replaces the line's earlier update, or adds it after the lines so far. */
export function mergeTranscript(
	lines: Transcript[],
	line: Transcript,
): Transcript[] {
	const index = lines.findIndex(({ id }) => id === line.id);
	if (index === -1) return [...lines, line];
	return lines.with(index, line);
}

/** Everything heard so far, including a line still being transcribed. */
export function dictatedText(lines: Transcript[]): string {
	return lines
		.map(({ text }) => text.trim())
		.filter(Boolean)
		.join(" ");
}
