import { expect, test } from "vitest";
import { addTalk, findTalk, search } from "./talks-fixtures";

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

test("a new talk goes live under an id made from its title", () => {
	const talk = addTalk({
		title: "Understanding Kubernetes",
		speaker: "Ada Lovelace",
		description: "",
	});

	expect(talk.id).toBe("understanding-kubernetes-2");
	expect(talk.status).toBe("live");
	expect(findTalk(talk.id)).toEqual(talk);
});
