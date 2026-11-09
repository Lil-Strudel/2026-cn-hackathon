import { expect, test } from "vitest";
import { splitHighlight } from "./highlight";

const parts = (fragment: string) =>
	splitHighlight(fragment).map(({ text, match }) => [text, match]);

test("marked terms become matches and the text around them stays plain", () => {
	expect(parts("run <mark>Kubernetes</mark> on <mark>Talos</mark>")).toEqual([
		["run ", false],
		["Kubernetes", true],
		[" on ", false],
		["Talos", true],
	]);
});

test("a fragment that starts with a match has no empty leading part", () => {
	expect(parts("<mark>eBPF</mark> everywhere")).toEqual([
		["eBPF", true],
		[" everywhere", false],
	]);
});

test("other markup is kept as text, never parsed", () => {
	expect(parts("<b>bold</b>")).toEqual([["<b>bold</b>", false]]);
});

test("offsets are unique so they can key the rendered parts", () => {
	const offsets = splitHighlight("<mark>a</mark>b<mark>c</mark>").map(
		(part) => part.offset,
	);
	expect(offsets).toEqual([0, 1, 2]);
});
