import workletUrl from "~/lib/pcm-encoder.worklet.ts?worker&url";

export type MicrophoneStream = { stop: () => void };

export type Dictation = MicrophoneStream & {
	/** Stops the microphone and resolves once the last words are transcribed. */
	finish: () => Promise<void>;
};

export type Transcript = { id: string; text: string; final: boolean };

const endOfAudio = "end";

/** Streams the microphone to caption a talk. */
export function streamMicrophone(
	talkId: string,
	onClose: (reason: string) => void,
): Promise<MicrophoneStream> {
	return openMicrophone(
		`/api/talks/${encodeURIComponent(talkId)}/audio`,
		onClose,
	);
}

/**
 * Streams the microphone to be transcribed without captioning a talk.
 * onTranscript gets every update to a line, partial and final.
 */
export function dictate(
	onTranscript: (transcript: Transcript) => void,
	onClose: (reason: string) => void,
): Promise<Dictation> {
	return openMicrophone("/api/transcribe", onClose, (data) =>
		onTranscript(JSON.parse(data)),
	);
}

/**
 * Streams the microphone to the api as PCM16 over a WebSocket. The
 * AudioContext runs at the device's own rate and the worklet resamples,
 * because Firefox refuses to connect a microphone to a context running at a
 * different rate. onClose fires when the connection ends without stop() or finish().
 */
async function openMicrophone(
	path: string,
	onClose: (reason: string) => void,
	onMessage?: (data: string) => void,
): Promise<Dictation> {
	const media = await navigator.mediaDevices.getUserMedia({
		audio: {
			channelCount: 1,
			echoCancellation: true,
			noiseSuppression: true,
			autoGainControl: true,
		},
	});
	const stopTracks = () => {
		for (const track of media.getTracks()) track.stop();
	};

	let socket: WebSocket;
	try {
		socket = await openSocket(path);
	} catch (error) {
		stopTracks();
		throw error;
	}

	const audio = new AudioContext();
	let stopped = false;
	const release = () => {
		stopped = true;
		stopTracks();
		void audio.close();
	};
	const stop = () => {
		if (stopped) return;
		release();
		if (socket.readyState === WebSocket.OPEN) socket.close(1000);
	};
	const finish = () =>
		new Promise<void>((resolve) => {
			if (stopped || socket.readyState !== WebSocket.OPEN) {
				stop();
				resolve();
				return;
			}
			release();
			socket.addEventListener("close", () => resolve(), { once: true });
			socket.send(endOfAudio);
		});

	socket.addEventListener("close", (event) => {
		if (stopped) return;
		stop();
		onClose(event.reason || "The connection to the server was lost.");
	});
	if (onMessage)
		socket.addEventListener("message", (event) => {
			if (typeof event.data === "string") onMessage(event.data);
		});

	try {
		await audio.audioWorklet.addModule(workletUrl);
		const encoder = new AudioWorkletNode(audio, "pcm-encoder", {
			numberOfOutputs: 0,
		});
		encoder.port.onmessage = (event: MessageEvent<ArrayBuffer>) => {
			if (!stopped && socket.readyState === WebSocket.OPEN)
				socket.send(event.data);
		};
		audio.createMediaStreamSource(media).connect(encoder);
	} catch (error) {
		stop();
		throw error;
	}

	return { stop, finish };
}

function openSocket(path: string): Promise<WebSocket> {
	const scheme = location.protocol === "https:" ? "wss" : "ws";
	const socket = new WebSocket(`${scheme}://${location.host}${path}`);
	socket.binaryType = "arraybuffer";
	return new Promise((resolve, reject) => {
		socket.addEventListener("open", () => resolve(socket), { once: true });
		socket.addEventListener(
			"close",
			() => reject(new Error("Could not connect to the caption server.")),
			{ once: true },
		);
	});
}

export function describeMicrophoneError(cause: unknown): string {
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
