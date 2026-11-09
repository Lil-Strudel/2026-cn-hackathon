import { Resampler, toPcm16, transcriptionSampleRate } from "~/lib/pcm";

// TypeScript's DOM lib has no AudioWorkletGlobalScope types.
declare const sampleRate: number;
declare class AudioWorkletProcessor {
	readonly port: MessagePort;
}
declare function registerProcessor(
	name: string,
	processor: new () => AudioWorkletProcessor,
): void;

const chunkSamples = transcriptionSampleRate / 10;

/**
 * Turns the microphone into 100 ms chunks of mono PCM16 at the rate the
 * transcriber expects, posted to the main thread as ArrayBuffers.
 */
class PcmEncoder extends AudioWorkletProcessor {
	private readonly resampler = new Resampler(
		sampleRate,
		transcriptionSampleRate,
	);
	private pending = new Float32Array(chunkSamples);
	private filled = 0;

	process(inputs: Float32Array[][]): boolean {
		const channel = inputs[0]?.[0];
		if (!channel) return true;

		let samples = this.resampler.process(channel);
		while (samples.length > 0) {
			const take = Math.min(samples.length, chunkSamples - this.filled);
			this.pending.set(samples.subarray(0, take), this.filled);
			this.filled += take;
			samples = samples.subarray(take);
			if (this.filled === chunkSamples) {
				const pcm = toPcm16(this.pending);
				this.port.postMessage(pcm.buffer, [pcm.buffer]);
				this.pending = new Float32Array(chunkSamples);
				this.filled = 0;
			}
		}
		return true;
	}
}

registerProcessor("pcm-encoder", PcmEncoder);
