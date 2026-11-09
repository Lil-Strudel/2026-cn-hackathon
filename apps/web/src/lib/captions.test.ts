import { expect, test } from "vitest";
import {
	type CaptionLine,
	heardText,
	mergeLines,
	splitCorrections,
} from "./captions";

const line = (overrides: Partial<CaptionLine>): CaptionLine => ({
	id: "1",
	at: "2026-11-09T14:00:00Z",
	text: "",
	final: true,
	corrections: [],
	...overrides,
});

test("corrected terms are split out with what was heard", () => {
	const parts = splitCorrections(
		line({
			text: "run kubectl apply against etcd",
			corrections: [
				{ start: 26, end: 30, heard: "at CD" },
				{ start: 4, end: 11, heard: "cube control" },
			],
		}),
	);
	expect(parts.map(({ text, heard }) => [text, heard])).toEqual([
		["run ", undefined],
		["kubectl", "cube control"],
		[" apply against ", undefined],
		["etcd", "at CD"],
	]);
});

test("corrections that overlap or run past the text are ignored", () => {
	const parts = splitCorrections(
		line({
			text: "hello world",
			corrections: [
				{ start: 0, end: 5, heard: "yellow" },
				{ start: 3, end: 8, heard: "overlap" },
				{ start: 6, end: 40, heard: "too long" },
			],
		}),
	);
	expect(parts.map(({ text }) => text)).toEqual(["hello", " world"]);
});

test("a partial line is replaced in place and lines stay in time order", () => {
	const first = line({ id: "a", at: "2026-11-09T14:00:01Z", text: "first" });
	const partial = line({
		id: "b",
		at: "2026-11-09T14:00:02Z",
		text: "sec",
		final: false,
	});
	const lines = mergeLines([partial], [first]);
	expect(lines.map((l) => l.id)).toEqual(["a", "b"]);

	const done = mergeLines(lines, [{ ...partial, text: "second", final: true }]);
	expect(done.map((l) => l.text)).toEqual(["first", "second"]);
});

test("a late partial never overwrites a finished line", () => {
	const done = line({ id: "a", text: "finished line" });
	const lines = mergeLines([done], [{ ...done, text: "fini", final: false }]);
	expect(lines[0]?.text).toBe("finished line");
});

test("the heard text puts back what Whisper heard in place of each correction", () => {
	expect(
		heardText(
			line({
				text: "run kubectl apply against etcd",
				corrections: [
					{ start: 26, end: 30, heard: "at CD" },
					{ start: 4, end: 11, heard: "cube control" },
				],
			}),
		),
	).toBe("run cube control apply against at CD");
});
