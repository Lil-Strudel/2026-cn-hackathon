import { expect, test } from "vitest";
import { search } from "./talks-fixtures";

test("live talks do not return transcript search hits", () => {
	expect(
		search("kubelet").some(
			({ talk }) => talk.id === "understanding-kubernetes",
		),
	).toBe(false);
});

test("finished talks remain searchable by transcript", () => {
	const hit = search("Friday afternoon").find(
		({ talk }) => talk.id === "gitops-without-tears",
	);

	expect(hit?.highlights).toContainEqual(
		expect.objectContaining({ field: "transcript" }),
	);
});
