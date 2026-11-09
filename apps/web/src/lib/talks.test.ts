import { afterEach, expect, test, vi } from "vitest";
import { createTalk, fetchTalks, searchTalks } from "./talks";

afterEach(() => vi.unstubAllGlobals());

function stubFetch(body: unknown, status = 200) {
	const fetch = vi.fn(async (_path: string, _init?: RequestInit) =>
		Response.json(body, { status }),
	);
	vi.stubGlobal("fetch", fetch);
	return fetch;
}

test("a new talk is created on the server, so everyone can open it", async () => {
	const created = { id: "guitar-demo", title: "Guitar demo" };
	const fetch = stubFetch(created, 201);
	const input = { title: "Guitar demo", speaker: "Aaron", description: "" };

	await expect(createTalk(input)).resolves.toEqual(created);
	expect(fetch).toHaveBeenCalledWith("/api/talks", {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(input),
	});
});

test("talks and search results come from the api", async () => {
	const fetch = stubFetch([]);
	await fetchTalks();
	await searchTalks("guitar & bass");
	expect(fetch.mock.calls.map(([path]) => path)).toEqual([
		"/api/talks",
		"/api/search?q=guitar+%26+bass",
	]);
});

test("a failed create reports the status", async () => {
	stubFetch("title must be 1 to 120 characters", 400);
	await expect(
		createTalk({ title: "", speaker: "Aaron", description: "" }),
	).rejects.toThrow("POST /api/talks: 400");
});
