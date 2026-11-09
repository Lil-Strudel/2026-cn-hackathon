import { fireEvent, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import * as api from "~/lib/api";
import { renderRoute } from "~/test/render-route";

vi.mock("~/lib/api", { spy: true });

beforeEach(() => {
	vi.mocked(api.listTerms).mockResolvedValue([
		{ term: "Gitar", context: ["Sonar", "review"] },
		{ term: "KEDA" },
	]);
});

afterEach(() => vi.restoreAllMocks());

async function vocabularyList() {
	const heading = await screen.findByRole("heading", { name: "Vocabulary" });
	return within(heading.closest("[data-slot=card]") as HTMLElement);
}

test("adds a term with its context words and refreshes the list", async () => {
	vi.mocked(api.addTerms).mockResolvedValue();
	renderRoute("/corrections");
	const list = await vocabularyList();
	await list.findByText("KEDA");

	vi.mocked(api.listTerms).mockResolvedValue([
		{ term: "Ferret", context: ["Linux"] },
	]);
	fireEvent.change(screen.getByLabelText("Term"), {
		target: { value: " Ferret " },
	});
	fireEvent.change(screen.getByLabelText("Context words (optional)"), {
		target: { value: "Linux, , " },
	});
	fireEvent.click(screen.getByRole("button", { name: "Add" }));

	expect(
		await screen.findByText("Added. Try it in the box above."),
	).toBeDefined();
	expect(api.addTerms).toHaveBeenCalledWith(
		[{ term: "Ferret", context: ["Linux"] }],
		expect.anything(),
	);
	expect(screen.getByLabelText("Context words (optional)")).toHaveProperty(
		"value",
		"",
	);
	expect(await list.findByText("Ferret")).toBeDefined();
});

test("lists terms with their context words", async () => {
	renderRoute("/corrections");
	const list = await vocabularyList();

	expect(await list.findByText("Gitar")).toBeDefined();
	expect(list.getByText("Sonar")).toBeDefined();
	expect(list.getByText("review")).toBeDefined();
	expect(list.getByRole("button", { name: "Remove KEDA" })).toBeDefined();
});

test("removes a term and refreshes the list", async () => {
	vi.mocked(api.removeTerm).mockResolvedValue();
	renderRoute("/corrections");
	const list = await vocabularyList();
	const remove = await list.findByRole("button", { name: "Remove Gitar" });

	vi.mocked(api.listTerms).mockResolvedValue([{ term: "KEDA" }]);
	fireEvent.click(remove);

	expect(await list.findByText("Removed Gitar.")).toBeDefined();
	expect(api.removeTerm).toHaveBeenCalledWith("Gitar", expect.anything());
	expect(list.queryByRole("button", { name: "Remove Gitar" })).toBeNull();
});

test("shows the error when terms fail to load", async () => {
	vi.mocked(api.listTerms).mockRejectedValue(
		new Error("GET /api/vocabulary: 502"),
	);
	renderRoute("/corrections");
	const list = await vocabularyList();

	expect(
		await list.findByText("Couldn't load terms: GET /api/vocabulary: 502"),
	).toBeDefined();
});
