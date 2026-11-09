export const captionViews = ["transcript", "large"] as const;
export type CaptionView = (typeof captionViews)[number];

const storageKey = "caption-view";

export function isCaptionView(value: unknown): value is CaptionView {
	return captionViews.includes(value as CaptionView);
}

export function readCaptionView(): CaptionView {
	try {
		const stored = localStorage.getItem(storageKey);
		return isCaptionView(stored) ? stored : "transcript";
	} catch {
		return "transcript";
	}
}

export function saveCaptionView(view: CaptionView) {
	try {
		localStorage.setItem(storageKey, view);
	} catch {}
}
