import { screen, within } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import * as talks from "~/lib/talks";
import { renderRoute } from "~/test/render-route";

vi.mock("~/lib/talks", { spy: true });

afterEach(() => vi.restoreAllMocks());

test("lists live talks apart from past ones", async () => {
	renderRoute("/");
	const live = (
		await screen.findByRole("heading", { name: "Live now" })
	).closest("section");
	if (!live) throw new Error("no live talks section");

	expect(
		await within(live).findByText("Understanding Kubernetes"),
	).toBeDefined();
	expect(within(live).queryByText("GitOps without tears")).toBeNull();
	expect(screen.getByText("GitOps without tears")).toBeDefined();
});

test("shows the error when talks fail to load", async () => {
	vi.mocked(talks.fetchTalks).mockRejectedValue(
		new Error("GET /api/talks: 502"),
	);
	renderRoute("/");
	expect(
		await screen.findByText("Couldn't load talks: GET /api/talks: 502"),
	).toBeDefined();
	expect(screen.getByRole("button", { name: "Try again" })).toBeDefined();
});
