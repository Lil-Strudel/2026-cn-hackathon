import { afterEach, expect, test } from "vitest";
import { readPreference, resolveTheme, savePreference } from "./theme";

afterEach(() => localStorage.clear());

test("system follows the operating system, explicit choices override it", () => {
	expect(resolveTheme("system", true)).toBe("dark");
	expect(resolveTheme("system", false)).toBe("light");
	expect(resolveTheme("light", true)).toBe("light");
	expect(resolveTheme("dark", false)).toBe("dark");
});

test("choosing system forgets the stored override", () => {
	savePreference("dark");
	expect(readPreference()).toBe("dark");
	savePreference("system");
	expect(localStorage.getItem("theme")).toBeNull();
	expect(readPreference()).toBe("system");
});

test("an unknown stored value falls back to system", () => {
	localStorage.setItem("theme", "sepia");
	expect(readPreference()).toBe("system");
});
