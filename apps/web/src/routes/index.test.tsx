import { screen, within } from "@testing-library/react";
import { expect, test } from "vitest";
import { renderRoute } from "~/test/render-route";

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
