import workletUrl from "~/lib/pcm-encoder.worklet.ts?worker&url";

export type MicrophoneStream = { stop: () => void };

/**
 * Streams the microphone to the api as PCM16 over a WebSocket. The
 * AudioContext runs at the device's own rate and the worklet resamples,
 * because Firefox refuses to connect a microphone to a context running at a
 * different rate. onClose fires when the connection ends without stop().
 */
export async function streamMicrophone(
	talkId: string,
	onClose: (reason: string) => void,
): Promise<MicrophoneStream> {
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
		socket = await openSocket(talkId);
	} catch (error) {
		stopTracks();
		throw error;
	}

	const audio = new AudioContext();
	let stopped = false;
	const stop = () => {
		if (stopped) return;
		stopped = true;
		stopTracks();
		void audio.close();
		if (socket.readyState === WebSocket.OPEN) socket.close(1000);
	};

	socket.addEventListener("close", (event) => {
		if (stopped) return;
		stop();
		onClose(event.reason || "The connection to the server was lost.");
	});

	try {
		await audio.audioWorklet.addModule(workletUrl);
		const encoder = new AudioWorkletNode(audio, "pcm-encoder", {
			numberOfOutputs: 0,
		});
		encoder.port.onmessage = (event: MessageEvent<ArrayBuffer>) => {
			if (socket.readyState === WebSocket.OPEN) socket.send(event.data);
		};
		audio.createMediaStreamSource(media).connect(encoder);
	} catch (error) {
		stop();
		throw error;
	}

	return { stop };
}

function openSocket(talkId: string): Promise<WebSocket> {
	const scheme = location.protocol === "https:" ? "wss" : "ws";
	const socket = new WebSocket(
		`${scheme}://${location.host}/api/talks/${encodeURIComponent(talkId)}/audio`,
	);
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
