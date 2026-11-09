import { expect, test } from "vitest";
import type { CaptionLine } from "./captions";
import { groupTranscript } from "./transcript";

const line = (id: string, seconds: number, final = true): CaptionLine => ({
	id,
	at: new Date(Date.UTC(2026, 10, 9, 14, 0, seconds)).toISOString(),
	text: id,
	final,
	corrections: [],
});

test("lines within a minute of a paragraph's start share it", () => {
	const paragraphs = groupTranscript([
		line("a", 0),
		line("b", 30),
		line("c", 59),
		line("d", 60),
		line("e", 100),
		line("f", 125),
	]);
	expect(paragraphs.map((p) => p.lines.map((l) => l.id))).toEqual([
		["a", "b", "c"],
		["d", "e"],
		["f"],
	]);
});

test("partial lines are left out of the transcript", () => {
	const paragraphs = groupTranscript([line("a", 0), line("b", 5, false)]);
	expect(paragraphs.flatMap((p) => p.lines.map((l) => l.id))).toEqual(["a"]);
});
