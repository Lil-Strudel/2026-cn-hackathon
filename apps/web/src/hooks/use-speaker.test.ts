import { act, renderHook } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { useSpeaker } from "~/hooks/use-speaker";
import * as microphone from "~/lib/microphone-stream";

vi.mock("~/lib/microphone-stream", { spy: true });

afterEach(() => vi.restoreAllMocks());

function pendingStream() {
	const stream = { stop: vi.fn() };
	let resolve: (value: typeof stream) => void = () => {};
	vi.mocked(microphone.streamMicrophone).mockReturnValue(
		new Promise((r) => {
			resolve = r;
		}),
	);
	return { stream, resolve: () => resolve(stream) };
}

test("leaving the page while the microphone starts stops it once it does", async () => {
	const { stream, resolve } = pendingStream();
	const { result, unmount } = renderHook(() => useSpeaker("demo"));

	let starting: Promise<void> = Promise.resolve();
	act(() => {
		starting = result.current.start();
	});
	unmount();
	resolve();
	await starting;

	expect(stream.stop).toHaveBeenCalledOnce();
});

test("a microphone that starts while the page is open stays on until stopped", async () => {
	const { stream, resolve } = pendingStream();
	const { result } = renderHook(() => useSpeaker("demo"));

	await act(async () => {
		const starting = result.current.start();
		resolve();
		await starting;
	});
	expect(result.current.status).toBe("live");
	expect(stream.stop).not.toHaveBeenCalled();

	act(() => result.current.stop());
	expect(stream.stop).toHaveBeenCalledOnce();
	expect(result.current.status).toBe("idle");
});
