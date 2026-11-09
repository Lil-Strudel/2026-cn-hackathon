import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import {
	type CaptionLine,
	fetchCaptions,
	mergeLines,
	type StreamStatus,
	subscribeCaptions,
} from "~/lib/captions";
import {
	recordCaptionsReconnecting,
	recordFirstCaption,
} from "~/lib/telemetry";

export function useCaptions(talkId: string) {
	const history = useQuery({
		queryKey: ["captions", talkId],
		queryFn: () => fetchCaptions(talkId),
	});
	const [live, setLive] = useState<CaptionLine[]>([]);
	const [status, setStatus] = useState<StreamStatus>("connecting");

	useEffect(() => {
		setLive([]);
		setStatus("connecting");
		const subscribedAt = performance.now();
		let waiting = true;

		return subscribeCaptions(
			talkId,
			(line) => {
				if (waiting) {
					waiting = false;
					recordFirstCaption(performance.now() - subscribedAt);
				}
				setLive((lines) => mergeLines(lines, [line]));
			},
			(next) => {
				if (next === "reconnecting") recordCaptionsReconnecting();
				setStatus(next);
			},
		);
	}, [talkId]);

	const lines = useMemo(
		() => mergeLines(history.data ?? [], live),
		[history.data, live],
	);

	return { lines, status, historyError: history.error };
}
