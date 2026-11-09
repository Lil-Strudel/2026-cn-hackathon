import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import * as captions from "~/lib/captions";
import { renderRoute } from "~/test/render-route";

vi.mock("~/lib/captions", { spy: true });

beforeEach(() => {
	vi.mocked(captions.fetchCaptions).mockResolvedValue([]);
	vi.mocked(captions.subscribeCaptions).mockReturnValue(() => {});
});

test("the home page offers a new talk", async () => {
	renderRoute("/");
	const link = await screen.findByRole("link", { name: "New talk" });
	expect(link.getAttribute("href")).toBe("/talks/new");
});

test("creating a talk goes live and opens captioning it", async () => {
	renderRoute("/talks/new");
	const create = await screen.findByRole("button", { name: "Create talk" });
	expect(create.hasAttribute("disabled")).toBe(true);

	fireEvent.change(screen.getByLabelText("Title"), {
		target: { value: "Demo day" },
	});
	fireEvent.change(screen.getByLabelText("Speaker"), {
		target: { value: "Sam Rivera" },
	});
	fireEvent.click(create);

	expect(
		await screen.findByRole("heading", { name: "Caption “Demo day”" }),
	).toBeDefined();
	expect(
		screen.getByRole("button", { name: "Start captioning" }),
	).toBeDefined();
});
