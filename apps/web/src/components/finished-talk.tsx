import { useQuery } from "@tanstack/react-query";
import { useLocation } from "@tanstack/react-router";
import { Sparkles } from "lucide-react";
import { type ReactNode, useEffect } from "react";
import { CaptionText } from "~/components/caption-text";
import { Card, CardContent, CardHeader, CardTitle } from "~/components/ui/card";
import { fetchCaptions } from "~/lib/captions";
import { formatClock } from "~/lib/format";
import { fetchNotes } from "~/lib/talks";
import { groupTranscript, lineAnchor } from "~/lib/transcript";

const notesPollMs = 15_000;

export function FinishedTalk({ talkId }: { talkId: string }) {
	const notes = useQuery({
		queryKey: ["notes", talkId],
		queryFn: () => fetchNotes(talkId),
		// The api writes notes once the captions have been quiet for a minute,
		// so keep asking until they exist.
		refetchInterval: (query) =>
			query.state.data?.summary === null ? notesPollMs : false,
	});

	return (
		<div className="grid items-start gap-6 lg:grid-cols-5">
			<div className="flex min-w-0 flex-col gap-6 lg:sticky lg:top-8 lg:col-span-2">
				<Card>
					<CardHeader>
						<CardTitle>
							<h2 className="text-lg">Notes</h2>
						</CardTitle>
						<GeneratedLabel />
					</CardHeader>
					<CardContent>
						{notes.isPending && <Muted>Loading notes…</Muted>}
						{notes.isError && <LoadError what="the notes" />}
						{notes.data?.notes === null && (
							<Muted>
								Notes are written from the transcript shortly after the talk
								ends.
							</Muted>
						)}
						{notes.data?.notes && (
							<ul className="flex list-disc flex-col gap-2 pl-5 text-base">
								{notes.data.notes.map((note) => (
									<li key={note}>{note}</li>
								))}
							</ul>
						)}
					</CardContent>
				</Card>

				<Card>
					<CardHeader>
						<CardTitle>
							<h2 className="text-lg">Summary</h2>
						</CardTitle>
						<GeneratedLabel />
					</CardHeader>
					<CardContent>
						{notes.isPending && <Muted>Loading summary…</Muted>}
						{notes.isError && <LoadError what="the summary" />}
						{notes.data?.summary === null && (
							<Muted>The summary is written shortly after the talk ends.</Muted>
						)}
						{notes.data?.summary && (
							<p className="text-base leading-relaxed">{notes.data.summary}</p>
						)}
					</CardContent>
				</Card>
			</div>

			<Transcript talkId={talkId} />
		</div>
	);
}

function Transcript({ talkId }: { talkId: string }) {
	const captions = useQuery({
		queryKey: ["captions", talkId],
		queryFn: () => fetchCaptions(talkId),
	});
	const { hash } = useLocation();
	const paragraphs = groupTranscript(captions.data ?? []);

	// The router scrolls to the hash before the transcript has loaded, so do it again
	// once it has, and move focus there so keyboard and screen reader users land on it too.
	useEffect(() => {
		if (!captions.data || !hash) return;
		const line = document.getElementById(hash);
		line?.scrollIntoView({ block: "center" });
		line?.focus({ preventScroll: true });
	}, [captions.data, hash]);

	return (
		<Card className="min-w-0 lg:col-span-3">
			<CardHeader>
				<CardTitle>
					<h2 className="text-lg">Transcript</h2>
				</CardTitle>
			</CardHeader>
			<CardContent className="flex flex-col gap-6">
				{captions.isPending && <Muted>Loading transcript…</Muted>}
				{captions.isError && <LoadError what="the transcript" />}
				{captions.data?.length === 0 && (
					<Muted>Nothing was captioned for this talk.</Muted>
				)}
				{paragraphs.map((paragraph) => (
					<div key={paragraph.at} className="flex flex-col gap-1">
						<time
							dateTime={paragraph.at}
							className="font-mono text-sm text-muted-foreground tabular-nums"
						>
							{formatClock(paragraph.at)}
						</time>
						<p className="max-w-[65ch] text-caption">
							{paragraph.lines.map((line) => (
								<span
									key={line.id}
									id={lineAnchor(line.id)}
									// pushState navigation doesn't update :target, so mark the linked line ourselves.
									data-linked={hash === lineAnchor(line.id) || undefined}
									tabIndex={hash === lineAnchor(line.id) ? -1 : undefined}
									className="rounded-sm outline-none data-linked:bg-highlight data-linked:text-highlight-foreground"
								>
									<CaptionText line={line} />{" "}
								</span>
							))}
						</p>
					</div>
				))}
			</CardContent>
		</Card>
	);
}

function GeneratedLabel() {
	return (
		<p className="flex items-center gap-1.5 text-sm text-muted-foreground">
			<Sparkles aria-hidden className="size-3.5" />
			Written by AI from the transcript
		</p>
	);
}

function Muted({ children }: { children: ReactNode }) {
	return <p className="text-sm text-muted-foreground">{children}</p>;
}

function LoadError({ what }: { what: string }) {
	return (
		<p role="alert" className="text-sm text-destructive">
			Couldn't load {what}. Reload the page to try again.
		</p>
	);
}
