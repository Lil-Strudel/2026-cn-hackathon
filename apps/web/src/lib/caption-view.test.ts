import { afterEach, expect, test } from "vitest";
import { readCaptionView, saveCaptionView } from "./caption-view";

afterEach(() => localStorage.clear());

test("the chosen view is remembered", () => {
	expect(readCaptionView()).toBe("transcript");
	saveCaptionView("large");
	expect(readCaptionView()).toBe("large");
});

test("an unknown stored value falls back to the transcript", () => {
	localStorage.setItem("caption-view", "huge");
	expect(readCaptionView()).toBe("transcript");
});
