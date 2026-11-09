import { act, fireEvent, screen } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import * as captions from "~/lib/captions";
import * as talks from "~/lib/talks";
import { renderRoute } from "~/test/render-route";
import { mockTalksApi } from "~/test/talks";

vi.mock("~/lib/captions", { spy: true });
vi.mock("~/lib/talks", { spy: true });

const firstLine: captions.CaptionLine = {
	id: "s1-item_1",
	at: "2026-11-09T21:00:05.000Z",
	text: "Welcome to the control plane.",
	final: true,
	corrections: [],
};

beforeEach(() => {
	mockTalksApi();
	vi.mocked(captions.fetchCaptions).mockResolvedValue([]);
	vi.mocked(captions.subscribeCaptions).mockReturnValue(() => {});
});

test("a live talk links to captioning it", async () => {
	renderRoute("/talks/understanding-kubernetes");
	const link = await screen.findByRole("link", { name: "Caption this talk" });
	expect(link.getAttribute("href")).toBe(
		"/talks/understanding-kubernetes/speak",
	);
});

test("a live talk can be ended once it has a captioned line", async () => {
	let onLine: (line: captions.CaptionLine) => void = () => {};
	vi.mocked(captions.subscribeCaptions).mockImplementation((_, next) => {
		onLine = next;
		return () => {};
	});
	renderRoute("/talks/understanding-kubernetes");
	await screen.findByRole("link", { name: "Caption this talk" });
	expect(screen.queryByRole("button", { name: "End talk" })).toBeNull();

	act(() => onLine({ ...firstLine, text: "Welcome to", final: false }));
	expect(screen.queryByRole("button", { name: "End talk" })).toBeNull();

	act(() => onLine(firstLine));
	expect(screen.getByRole("button", { name: "End talk" })).toBeDefined();
});

test("ending a live talk turns it into a past talk", async () => {
	vi.mocked(captions.fetchCaptions).mockResolvedValue([firstLine]);
	renderRoute("/talks/understanding-kubernetes");
	fireEvent.click(await screen.findByRole("button", { name: "End talk" }));

	expect(await screen.findByRole("heading", { name: "Notes" })).toBeDefined();
	expect(talks.endTalk).toHaveBeenCalledWith("understanding-kubernetes");
	expect(screen.queryByRole("link", { name: "Caption this talk" })).toBeNull();
	expect(screen.queryByRole("button", { name: "End talk" })).toBeNull();
	expect(screen.getByRole("status").textContent).toBe("This talk has ended.");
	expect(document.activeElement).toBe(
		screen.getByRole("heading", { level: 1 }),
	);
});

test("a talk that fails to end says why and stays live", async () => {
	vi.mocked(talks.endTalk).mockRejectedValue(
		new Error("POST /api/talks/understanding-kubernetes/end: 502"),
	);
	vi.mocked(captions.fetchCaptions).mockResolvedValue([firstLine]);
	renderRoute("/talks/understanding-kubernetes");
	fireEvent.click(await screen.findByRole("button", { name: "End talk" }));

	expect((await screen.findByRole("alert")).textContent).toBe(
		"Couldn't end the talk: POST /api/talks/understanding-kubernetes/end: 502",
	);
	expect(screen.getByRole("link", { name: "Caption this talk" })).toBeDefined();
});

test("the speak page offers the microphone and shows the live captions", async () => {
	renderRoute("/talks/understanding-kubernetes/speak");
	expect(
		await screen.findByRole("heading", {
			name: "Caption “Understanding Kubernetes”",
		}),
	).toBeDefined();
	expect(
		screen.getByRole("button", { name: "Start captioning" }),
	).toBeDefined();
	expect(screen.getByText("Your microphone is off.")).toBeDefined();
	expect(screen.getByRole("heading", { name: "Live captions" })).toBeDefined();
});

test("a finished talk can't be captioned", async () => {
	renderRoute("/talks/gitops-without-tears/speak");
	expect(
		await screen.findByText(
			"This talk has finished, so it can't be captioned any more.",
		),
	).toBeDefined();
	expect(screen.queryByRole("button", { name: "Start captioning" })).toBeNull();
});
