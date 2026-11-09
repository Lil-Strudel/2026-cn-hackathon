import { screen } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import * as captions from "~/lib/captions";
import { renderRoute } from "~/test/render-route";

vi.mock("~/lib/captions", { spy: true });

beforeEach(() => {
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
