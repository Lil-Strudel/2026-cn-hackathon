import { describe, expect, it } from "vitest";
import { Resampler, toPcm16 } from "~/lib/pcm";

describe("Resampler", () => {
	it("halves 48 kHz to 24 kHz across block boundaries", () => {
		const resampler = new Resampler(48000, 24000);
		const ramp = Float32Array.from({ length: 256 }, (_, i) => i);
		const out = [
			...resampler.process(ramp.subarray(0, 128)),
			...resampler.process(ramp.subarray(128)),
		];
		expect(out).toHaveLength(128);
		out.forEach((v, i) => {
			expect(v).toBeCloseTo(i * 2);
		});
	});

	it("keeps the output length in proportion for uneven ratios", () => {
		const resampler = new Resampler(44100, 24000);
		let total = 0;
		for (let block = 0; block < 1000; block++) {
			total += resampler.process(new Float32Array(128)).length;
		}
		expect(Math.abs(total - (128_000 * 24000) / 44100)).toBeLessThanOrEqual(1);
	});

	it("interpolates between the previous block's last sample and the next block", () => {
		const resampler = new Resampler(3, 2);
		const first = resampler.process(Float32Array.from([0, 3, 6]));
		const second = resampler.process(Float32Array.from([9, 12, 15]));
		expect([...first, ...second]).toEqual([0, 4.5, 9, 13.5]);
	});
});

describe("toPcm16", () => {
	it("scales and clamps to signed 16-bit", () => {
		expect([...toPcm16(Float32Array.from([0, 1, -1, 2, -2, 0.5]))]).toEqual([
			0, 32767, -32768, 32767, -32768, 16383,
		]);
	});
});
