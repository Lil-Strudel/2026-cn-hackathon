import { expect, test } from "vitest";
import { talkSlug } from "./talks";

test("a talk's slug is its title in lowercase words joined by dashes", () => {
	expect(talkSlug("  Café: eBPF & You! ", new Set())).toBe("cafe-ebpf-you");
});

test("a slug already taken gets a number", () => {
	expect(talkSlug("Intro", new Set(["intro", "intro-2"]))).toBe("intro-3");
});

test("a title without letters or digits still gets a slug", () => {
	expect(talkSlug("🚀🚀", new Set())).toBe("talk");
});
