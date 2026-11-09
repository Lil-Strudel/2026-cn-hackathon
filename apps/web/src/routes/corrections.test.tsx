import { act, fireEvent, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import * as api from "~/lib/api";
import * as microphone from "~/lib/microphone-stream";
import { renderRoute } from "~/test/render-route";

vi.mock("~/lib/microphone-stream", { spy: true });
vi.mock("~/lib/api", { spy: true });

beforeEach(() => {
	vi.mocked(api.listTerms).mockResolvedValue([
		{ term: "Gitar", except: ["play", "country"] },
		{ term: "KEDA" },
	]);
});

afterEach(() => vi.restoreAllMocks());

async function vocabularyList() {
	const heading = await screen.findByRole("heading", { name: "Vocabulary" });
	return within(heading.closest("[data-slot=card]") as HTMLElement);
}

test("adds a term with its exception words and refreshes the list", async () => {
	vi.mocked(api.addTerms).mockResolvedValue();
	renderRoute("/corrections");
	const list = await vocabularyList();
	await list.findByText("KEDA");

	vi.mocked(api.listTerms).mockResolvedValue([
		{ term: "Ferret", except: ["pet"] },
	]);
	fireEvent.change(screen.getByLabelText("Term"), {
		target: { value: " Ferret " },
	});
	fireEvent.change(screen.getByLabelText("Except near (optional)"), {
		target: { value: "pet, , " },
	});
	fireEvent.click(screen.getByRole("button", { name: "Add" }));

	expect(
		await screen.findByText("Added. Try it in the box above."),
	).toBeDefined();
	expect(api.addTerms).toHaveBeenCalledWith(
		[{ term: "Ferret", except: ["pet"] }],
		expect.anything(),
	);
	expect(screen.getByLabelText("Except near (optional)")).toHaveProperty(
		"value",
		"",
	);
	expect(await list.findByText("Ferret")).toBeDefined();
});

test("lists terms with their exception words", async () => {
	renderRoute("/corrections");
	const list = await vocabularyList();

	expect(await list.findByText("Gitar")).toBeDefined();
	expect(list.getByText("play")).toBeDefined();
	expect(list.getByText("country")).toBeDefined();
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

test("what is said into the microphone is corrected when it stops", async () => {
	let send: (line: microphone.Transcript) => void = () => {};
	vi.mocked(microphone.dictate).mockImplementation(async (onTranscript) => {
		send = onTranscript;
		return { stop: vi.fn(), finish: vi.fn().mockResolvedValue(undefined) };
	});
	vi.mocked(api.correctText).mockResolvedValue({
		text: "over Cilium",
		segments: [{ text: "over " }, { text: "Cilium", heard: "silly um" }],
	});
	renderRoute("/corrections");

	fireEvent.click(await screen.findByRole("button", { name: "Speak" }));
	const stop = await screen.findByRole("button", { name: "Stop and correct" });
	act(() => send({ id: "a", text: "over silly um", final: true }));
	expect(
		screen.getByLabelText<HTMLTextAreaElement>("What was heard").value,
	).toBe("over silly um");

	fireEvent.click(stop);

	expect(await screen.findByText("Cilium")).toBeDefined();
	expect(vi.mocked(api.correctText).mock.calls[0]?.[0]).toBe("over silly um");
});
