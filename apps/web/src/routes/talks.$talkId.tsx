import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, CircleStop, Mic } from "lucide-react";
import { useRef } from "react";
import { FinishedTalk } from "~/components/finished-talk";
import { LiveBadge } from "~/components/live-badge";
import { LiveCaptions } from "~/components/live-captions";
import { Button } from "~/components/ui/button";
import { useCaptions } from "~/hooks/use-captions";
import { formatDateTime, formatTime } from "~/lib/format";
import { endTalk, fetchTalk } from "~/lib/talks";
import { recordTalkEnded } from "~/lib/telemetry";

export const Route = createFileRoute("/talks/$talkId")({
	component: TalkPage,
});

function TalkPage() {
	const { talkId } = Route.useParams();
	const talk = useQuery({
		queryKey: ["talk", talkId],
		queryFn: () => fetchTalk(talkId),
	});
	const live = talk.data?.status === "live";
	const captions = useCaptions(talkId, live);
	// The api only ends a talk with stored captions, and only final lines are stored.
	const captioned = captions.lines.some((line) => line.final);
	const title = useRef<HTMLHeadingElement>(null);
	const queryClient = useQueryClient();

	const ending = useMutation({
		mutationFn: () => endTalk(talkId),
		onSuccess: (ended) => {
			recordTalkEnded(ended.id);
			queryClient.setQueryData(["talk", talkId], ended);
			void queryClient.invalidateQueries({ queryKey: ["talks"] });
			// The button that had focus is gone once the talk has ended.
			title.current?.focus();
		},
	});

	return (
		<main className="mx-auto flex max-w-5xl flex-col gap-6 p-4 sm:p-8">
			<Button asChild variant="ghost" size="sm" className="self-start">
				<Link to="/">
					<ArrowLeft aria-hidden />
					All talks
				</Link>
			</Button>

			{talk.isPending && (
				<p className="text-sm text-muted-foreground">Loading talk…</p>
			)}
			{talk.isError && (
				<p role="alert" className="text-sm text-destructive">
					Couldn't load this talk: {talk.error.message}
				</p>
			)}
			{talk.data && (
				<header className="flex flex-col gap-2">
					<div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground">
						{talk.data.status === "live" && <LiveBadge />}
						<span>{talk.data.speaker}</span>
						<time dateTime={talk.data.startedAt} className="tabular-nums">
							{talk.data.status === "live"
								? `Started ${formatTime(talk.data.startedAt)}`
								: formatDateTime(talk.data.startedAt)}
						</time>
					</div>
					<h1
						ref={title}
						tabIndex={-1}
						className="text-2xl font-semibold tracking-tight"
					>
						{talk.data.title}
					</h1>
					<p className="max-w-[65ch] text-sm text-muted-foreground">
						{talk.data.description}
					</p>
					{talk.data.status === "live" && (
						<div className="flex flex-wrap gap-2">
							<Button asChild variant="outline" size="sm">
								<Link to="/talks/$talkId/speak" params={{ talkId }}>
									<Mic aria-hidden />
									Caption this talk
								</Link>
							</Button>
							{captioned && (
								<Button
									variant="outline"
									size="sm"
									disabled={ending.isPending}
									onClick={() => ending.mutate()}
								>
									<CircleStop aria-hidden />
									{ending.isPending ? "Ending…" : "End talk"}
								</Button>
							)}
						</div>
					)}
					{ending.isError && (
						<p role="alert" className="text-sm text-destructive">
							Couldn't end the talk: {ending.error.message}
						</p>
					)}
					<p role="status" className="sr-only">
						{ending.isSuccess && "This talk has ended."}
					</p>
				</header>
			)}
			{live && <LiveCaptions captions={captions} />}
			{talk.data?.status === "finished" && <FinishedTalk talkId={talkId} />}
		</main>
	);
}
