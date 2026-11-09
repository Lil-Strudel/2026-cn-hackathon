export const transcriptionSampleRate = 24000;

/**
 * Streams audio from one sample rate to a lower one by linear interpolation,
 * carrying its position across blocks so block edges leave no clicks. Speech
 * has little energy above the 12 kHz the output can hold, so aliasing is
 * not worth a low-pass filter here.
 */
export class Resampler {
	private readonly step: number;
	private position = 0;
	private previous = 0;

	constructor(fromRate: number, toRate: number) {
		this.step = fromRate / toRate;
	}

	process(input: Float32Array): Float32Array {
		const output: number[] = [];
		// position is relative to input[0]; -1 refers to the previous block's last sample.
		while (this.position < input.length - 1) {
			const index = Math.floor(this.position);
			const fraction = this.position - index;
			const a = index < 0 ? this.previous : (input[index] ?? 0);
			const b = input[index + 1] ?? 0;
			output.push(a + (b - a) * fraction);
			this.position += this.step;
		}
		this.position -= input.length;
		this.previous = input[input.length - 1] ?? this.previous;
		return Float32Array.from(output);
	}
}

export function toPcm16(samples: Float32Array): Int16Array {
	const pcm = new Int16Array(samples.length);
	for (let i = 0; i < samples.length; i++) {
		const s = Math.max(-1, Math.min(1, samples[i] ?? 0));
		pcm[i] = s < 0 ? s * 0x8000 : s * 0x7fff;
	}
	return pcm;
}
