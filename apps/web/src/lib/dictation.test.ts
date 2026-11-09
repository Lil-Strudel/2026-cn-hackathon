import { expect, test } from "vitest";
import { dictatedText, mergeTranscript } from "./dictation";

test("a line's updates replace each other in place", () => {
	let lines = mergeTranscript([], { id: "a", text: "our pods", final: false });
	lines = mergeTranscript(lines, { id: "b", text: "talk", final: false });
	lines = mergeTranscript(lines, {
		id: "a",
		text: "our pods talk over silly um",
		final: true,
	});

	expect(dictatedText(lines)).toBe("our pods talk over silly um talk");
});

test("empty updates leave no extra spaces", () => {
	expect(
		dictatedText([
			{ id: "a", text: " hello ", final: true },
			{ id: "b", text: "", final: false },
		]),
	).toBe("hello");
});
