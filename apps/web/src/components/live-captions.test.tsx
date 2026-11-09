import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { LiveCaptions } from "~/components/live-captions";
import * as captions from "~/lib/captions";

vi.mock("~/lib/captions", { spy: true });

beforeEach(() => {
	vi.mocked(captions.fetchCaptions).mockResolvedValue([
		{
			id: "1",
			at: "2026-11-09T14:00:00Z",
			text: "Our pods talk over Cilium.",
			final: true,
			corrections: [{ start: 19, end: 25, heard: "silly um" }],
		},
	]);
	vi.mocked(captions.subscribeCaptions).mockReturnValue(() => {});
});

test("turning correction off shows the captions as they were heard", async () => {
	render(
		<QueryClientProvider client={new QueryClient()}>
			<LiveCaptions talkId="demo" />
		</QueryClientProvider>,
	);
	expect(await screen.findByText("Cilium")).toBeDefined();

	fireEvent.click(screen.getByRole("switch", { name: "Correct terms" }));

	const log = screen.getByRole("log", { name: "Captions" });
	expect(log.textContent).toBe("Our pods talk over silly um.");

	fireEvent.click(screen.getByRole("switch", { name: "Correct terms" }));
	expect(screen.getByText("Cilium").tagName).toBe("MARK");
});
