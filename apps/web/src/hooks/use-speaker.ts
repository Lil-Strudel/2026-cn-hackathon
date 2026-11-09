import { useEffect, useRef, useState } from "react";
import {
	type MicrophoneStream,
	streamMicrophone,
} from "~/lib/microphone-stream";
import { recordSpeaker } from "~/lib/telemetry";

export type SpeakerStatus = "idle" | "starting" | "live";

export function useSpeaker(talkId: string) {
	const [status, setStatus] = useState<SpeakerStatus>("idle");
	const [error, setError] = useState<string>();
	const stream = useRef<MicrophoneStream>(undefined);
	const mounted = useRef(false);

	// Set here rather than at creation, because StrictMode unmounts and
	// remounts once in development.
	useEffect(() => {
		mounted.current = true;
		return () => {
			mounted.current = false;
			stream.current?.stop();
			stream.current = undefined;
		};
	}, []);

	async function start() {
		setError(undefined);
		setStatus("starting");
		try {
			const started = await streamMicrophone(talkId, (reason) => {
				stream.current = undefined;
				setStatus("idle");
				setError(reason);
				recordSpeaker(talkId, "failed", reason);
			});
			// Starting waits on the permission prompt and the socket, and the
			// speaker may have left the page meanwhile.
			if (!mounted.current) {
				started.stop();
				return;
			}
			stream.current = started;
			setStatus("live");
			recordSpeaker(talkId, "started");
		} catch (cause) {
			if (!mounted.current) return;
			const message = describeStartError(cause);
			setStatus("idle");
			setError(message);
			recordSpeaker(talkId, "failed", message);
		}
	}

	function stop() {
		stream.current?.stop();
		stream.current = undefined;
		setStatus("idle");
		recordSpeaker(talkId, "stopped");
	}

	return { status, error, start, stop };
}

function describeStartError(cause: unknown): string {
	if (cause instanceof DOMException && cause.name === "NotAllowedError") {
		return "Microphone access was blocked. Allow it in your browser's site settings and try again.";
	}
	if (cause instanceof DOMException && cause.name === "NotFoundError") {
		return "No microphone was found.";
	}
	return cause instanceof Error
		? cause.message
		: "Could not start the microphone.";
}
