import { useState } from "react";
import {
	type CaptionView,
	readCaptionView,
	saveCaptionView,
} from "~/lib/caption-view";
import { recordCaptionView } from "~/lib/telemetry";

export function useCaptionView() {
	const [view, setView] = useState(readCaptionView);

	function changeView(next: CaptionView) {
		saveCaptionView(next);
		setView(next);
		recordCaptionView(next);
	}

	return [view, changeView] as const;
}
