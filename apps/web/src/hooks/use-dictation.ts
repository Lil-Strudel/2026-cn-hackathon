import { useEffect, useRef, useState } from "react";
import { dictatedText, mergeTranscript } from "~/lib/dictation";
import {
	type Dictation,
	describeMicrophoneError,
	dictate,
	type Transcript,
} from "~/lib/microphone-stream";
import { recordDictation } from "~/lib/telemetry";

export type DictationStatus = "idle" | "starting" | "live" | "finishing";

export function useDictation() {
	const [status, setStatus] = useState<DictationStatus>("idle");
	const [error, setError] = useState<string>();
	const [lines, setLines] = useState<Transcript[]>([]);
	// finish() reads the lines after awaiting the last ones, past any render.
	const linesRef = useRef<Transcript[]>([]);
	const stream = useRef<Dictation>(undefined);
	const mounted = useRef(false);

	useEffect(() => {
		mounted.current = true;
		return () => {
			mounted.current = false;
			stream.current?.stop();
			stream.current = undefined;
		};
	}, []);

	function updateLines(next: Transcript[]) {
		linesRef.current = next;
		setLines(next);
	}

	async function start() {
		setError(undefined);
		updateLines([]);
		setStatus("starting");
		try {
			const started = await dictate(
				(line) => updateLines(mergeTranscript(linesRef.current, line)),
				(reason) => {
					stream.current = undefined;
					setStatus("idle");
					setError(reason);
					recordDictation("failed", reason);
				},
			);
			if (!mounted.current) {
				started.stop();
				return;
			}
			stream.current = started;
			setStatus("live");
			recordDictation("started");
		} catch (cause) {
			if (!mounted.current) return;
			const message = describeMicrophoneError(cause);
			setStatus("idle");
			setError(message);
			recordDictation("failed", message);
		}
	}

	/** Stops listening and resolves with everything heard. */
	async function finish(): Promise<string> {
		const current = stream.current;
		stream.current = undefined;
		setStatus("finishing");
		await current?.finish();
		if (mounted.current) setStatus("idle");
		recordDictation("finished");
		return dictatedText(linesRef.current);
	}

	return { status, error, text: dictatedText(lines), start, finish };
}
