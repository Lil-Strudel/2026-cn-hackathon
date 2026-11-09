import { ArrowDown, Captions, ScrollText } from "lucide-react";
import { useEffect, useId, useLayoutEffect, useRef, useState } from "react";
import { CaptionText } from "~/components/caption-text";
import { Button } from "~/components/ui/button";
import { Card } from "~/components/ui/card";
import { Label } from "~/components/ui/label";
import { Switch } from "~/components/ui/switch";
import { ToggleGroup, ToggleGroupItem } from "~/components/ui/toggle-group";
import { useCaptionView } from "~/hooks/use-caption-view";
import type { CaptionStream } from "~/hooks/use-captions";
import { type CaptionView, isCaptionView } from "~/lib/caption-view";
import type { CaptionLine } from "~/lib/captions";
import { recordCorrectionsShown } from "~/lib/telemetry";
import { cn } from "~/lib/utils";

const statusText = {
	connecting: "Connecting to the captions…",
	open: null,
	reconnecting: "Connection lost. Reconnecting…",
	closed: "Connection closed.",
};

export function LiveCaptions({ captions }: { captions: CaptionStream }) {
	const { lines, status, historyError } = captions;
	const [view, setView] = useCaptionView();
	const [corrected, setCorrected] = useState(true);
	const finished = lines.filter((line) => line.final);
	const partial = lines.findLast((line) => !line.final);
	const pinned = usePinnedToBottom(lines, view === "transcript");

	return (
		<section aria-labelledby="captions-heading" className="flex flex-col gap-4">
			<div className="flex flex-wrap items-center justify-between gap-4">
				<h2 id="captions-heading" className="text-lg font-semibold">
					Live captions
				</h2>
				<div className="flex flex-wrap items-center gap-4">
					<CorrectionSwitch
						corrected={corrected}
						onChange={(next) => {
							setCorrected(next);
							recordCorrectionsShown(next);
						}}
					/>
					<ViewToggle view={view} onChange={setView} />
				</div>
			</div>

			<p role="status" className="text-sm text-muted-foreground empty:hidden">
				{[
					statusText[status],
					historyError && "Earlier captions couldn't be loaded.",
				]
					.filter(Boolean)
					.join(" ")}
			</p>

			{view === "transcript" ? (
				<Transcript
					finished={finished}
					partial={partial}
					corrected={corrected}
				/>
			) : (
				<LargeCaptions
					finished={finished}
					partial={partial}
					corrected={corrected}
				/>
			)}

			{view === "transcript" && !pinned.atBottom && (
				<Button
					size="sm"
					className="fixed right-4 bottom-4 shadow-sm sm:right-8 sm:bottom-8"
					onClick={pinned.scrollToBottom}
				>
					<ArrowDown aria-hidden />
					Jump to latest
				</Button>
			)}
		</section>
	);
}

function CorrectionSwitch({
	corrected,
	onChange,
}: {
	corrected: boolean;
	onChange: (corrected: boolean) => void;
}) {
	const id = useId();
	return (
		<div className="flex items-center gap-2">
			<Switch id={id} checked={corrected} onCheckedChange={onChange} />
			<Label htmlFor={id}>Correct terms</Label>
		</div>
	);
}

function ViewToggle({
	view,
	onChange,
}: {
	view: CaptionView;
	onChange: (view: CaptionView) => void;
}) {
	return (
		<ToggleGroup
			type="single"
			variant="outline"
			size="sm"
			aria-label="Caption view"
			value={view}
			onValueChange={(value) => {
				if (isCaptionView(value)) onChange(value);
			}}
		>
			<ToggleGroupItem value="transcript">
				<ScrollText aria-hidden />
				Transcript
			</ToggleGroupItem>
			<ToggleGroupItem value="large">
				<Captions aria-hidden />
				Large captions
			</ToggleGroupItem>
		</ToggleGroup>
	);
}

type CaptionsProps = {
	finished: CaptionLine[];
	partial: CaptionLine | undefined;
	corrected: boolean;
};

// Only finished lines sit inside the live region, so screen readers announce whole
// sentences instead of every word as it is recognised. The region is keyed on
// `corrected` so flipping the switch swaps it whole, rather than rewriting every
// line in place, which a screen reader would read out again.
function Transcript({ finished, partial, corrected }: CaptionsProps) {
	return (
		<Card className="px-6 py-6">
			<div className="flex max-w-[65ch] flex-col gap-3 text-caption">
				<div
					key={String(corrected)}
					role="log"
					aria-label="Captions"
					className="flex flex-col gap-3"
				>
					{finished.map((line) => (
						<p key={line.id}>
							<CaptionText line={line} corrected={corrected} />
						</p>
					))}
				</div>
				{partial && (
					<p>
						<CaptionText line={partial} corrected={corrected} />
					</p>
				)}
				{finished.length === 0 && !partial && (
					<p className="text-base text-muted-foreground">
						Captions will appear here when the speaker starts talking.
					</p>
				)}
			</div>
		</Card>
	);
}

const largeLines = 2;

function LargeCaptions({ finished, partial, corrected }: CaptionsProps) {
	const recent = finished.slice(partial ? -(largeLines - 1) : -largeLines);

	return (
		<Card className="min-h-96 justify-center px-6 py-10 sm:px-12">
			<div className="mx-auto flex max-w-4xl flex-col gap-6 text-center text-3xl leading-snug sm:text-4xl">
				<div
					key={String(corrected)}
					role="log"
					aria-label="Captions"
					className="flex flex-col gap-6"
				>
					{recent.map((line, index) => (
						<p
							key={line.id}
							className={cn(
								(partial || index < recent.length - 1) &&
									"text-muted-foreground",
							)}
						>
							<CaptionText line={line} corrected={corrected} />
						</p>
					))}
				</div>
				{partial && (
					<p>
						<CaptionText line={partial} corrected={corrected} />
					</p>
				)}
				{recent.length === 0 && !partial && (
					<p className="text-xl text-muted-foreground">
						Captions will appear here when the speaker starts talking.
					</p>
				)}
			</div>
		</Card>
	);
}

const bottomSlackPx = 80;

function isAtBottom() {
	return (
		window.innerHeight + window.scrollY >=
		document.documentElement.scrollHeight - bottomSlackPx
	);
}

// Follows new captions like a terminal: stays at the bottom unless the reader
// scrolls up to re-read something, then waits until they come back.
function usePinnedToBottom(lines: CaptionLine[], follow: boolean) {
	const [atBottom, setAtBottom] = useState(true);
	const atBottomRef = useRef(true);

	useEffect(() => {
		const update = () => {
			atBottomRef.current = isAtBottom();
			setAtBottom(atBottomRef.current);
		};
		window.addEventListener("scroll", update, { passive: true });
		return () => window.removeEventListener("scroll", update);
	}, []);

	useLayoutEffect(() => {
		if (follow && lines.length > 0 && atBottomRef.current)
			window.scrollTo({ top: document.documentElement.scrollHeight });
	}, [lines, follow]);

	return {
		atBottom,
		scrollToBottom: () =>
			window.scrollTo({ top: document.documentElement.scrollHeight }),
	};
}
