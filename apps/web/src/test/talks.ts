import { vi } from "vitest";
import type { NewTalk, Talk } from "~/lib/talks";
import * as talks from "~/lib/talks";

export const liveTalk: Talk = {
	id: "understanding-kubernetes",
	title: "Understanding Kubernetes",
	speaker: "Priya Natarajan",
	description: "A tour of the control plane.",
	status: "live",
	startedAt: "2026-11-09T21:00:00.000Z",
	endedAt: null,
};

export const finishedTalk: Talk = {
	id: "gitops-without-tears",
	title: "GitOps without tears",
	speaker: "Lena Hoffmann",
	description: "Flux, OCI artifacts and cosign.",
	status: "finished",
	startedAt: "2026-11-09T18:00:00.000Z",
	endedAt: "2026-11-09T18:40:00.000Z",
};

/** Stands in for the talks api; call it in a file that has `vi.mock("~/lib/talks", { spy: true })`. */
export function mockTalksApi(stored: Talk[] = [liveTalk, finishedTalk]) {
	const all = [...stored];
	vi.mocked(talks.fetchTalks).mockImplementation(async () => all);
	vi.mocked(talks.fetchTalk).mockImplementation(async (id) => {
		const talk = all.find((t) => t.id === id);
		if (!talk) throw new Error(`GET /api/talks/${id}: 404`);
		return talk;
	});
	vi.mocked(talks.fetchNotes).mockResolvedValue({ summary: null, notes: null });
	vi.mocked(talks.searchTalks).mockResolvedValue([]);
	vi.mocked(talks.createTalk).mockImplementation(async (input: NewTalk) => {
		const talk: Talk = {
			...input,
			id: input.title.toLowerCase().replace(/[^a-z0-9]+/g, "-"),
			status: "live",
			startedAt: new Date().toISOString(),
			endedAt: null,
		};
		all.unshift(talk);
		return talk;
	});
	vi.mocked(talks.endTalk).mockImplementation(async (id) => {
		const talk = all.find((t) => t.id === id);
		if (!talk) throw new Error(`POST /api/talks/${id}/end: 404`);
		const ended: Talk = {
			...talk,
			status: "finished",
			endedAt: talk.endedAt ?? new Date().toISOString(),
		};
		all.splice(all.indexOf(talk), 1, ended);
		return ended;
	});
}
