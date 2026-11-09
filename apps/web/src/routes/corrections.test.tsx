import { fireEvent, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import * as api from "~/lib/api";
import { renderRoute } from "~/test/render-route";

vi.mock("~/lib/api", { spy: true });

afterEach(() => vi.restoreAllMocks());

test("adds a term with its context words", async () => {
	vi.mocked(api.addTerms).mockResolvedValue();
	renderRoute("/corrections");

	fireEvent.change(await screen.findByLabelText("Term"), {
		target: { value: " Gitar " },
	});
	fireEvent.change(screen.getByLabelText("Context words (optional)"), {
		target: { value: "Sonar, , pull request ," },
	});
	fireEvent.click(screen.getByRole("button", { name: "Add" }));

	expect(
		await screen.findByText("Added. Try it in the box above."),
	).toBeDefined();
	expect(api.addTerms).toHaveBeenCalledWith(
		[{ term: "Gitar", context: ["Sonar", "pull request"] }],
		expect.anything(),
	);
	expect(screen.getByLabelText("Context words (optional)")).toHaveProperty(
		"value",
		"",
	);
});
