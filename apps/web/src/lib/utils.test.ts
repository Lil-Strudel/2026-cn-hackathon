import { expect, test } from "vitest";
import { cn } from "./utils";

test("cn lets later tailwind classes win", () => {
	expect(cn("p-2 text-sm", "p-4")).toBe("text-sm p-4");
});
