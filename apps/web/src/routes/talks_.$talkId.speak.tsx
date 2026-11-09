import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, Mic, MicOff } from "lucide-react";
import { LiveBadge } from "~/components/live-badge";
import { LiveCaptions } from "~/components/live-captions";
import { Button } from "~/components/ui/button";
import { useSpeaker } from "~/hooks/use-speaker";
import { fetchTalk } from "~/lib/talks";

export const Route = createFileRoute("/talks_/$talkId/speak")({
	component: SpeakPage,
});

const speakerStatusText = {
	idle: "Your microphone is off.",
	starting: "Starting your microphone…",
	live: "Your microphone is on. Everything you say is captioned.",
} as const;

function SpeakPage() {
	const { talkId } = Route.useParams();
	const talk = useQuery({
		queryKey: ["talk", talkId],
		queryFn: () => fetchTalk(talkId),
	});

	return (
		<main className="mx-auto flex max-w-5xl flex-col gap-6 p-4 sm:p-8">
			<Button asChild variant="ghost" size="sm" className="self-start">
				<Link to="/talks/$talkId" params={{ talkId }}>
					<ArrowLeft aria-hidden />
					Back to the talk
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
					</div>
					<h1 className="text-2xl font-semibold tracking-tight">
						Caption “{talk.data.title}”
					</h1>
				</header>
			)}
			{talk.data?.status === "live" && <SpeakerControls talkId={talkId} />}
			{talk.data?.status === "finished" && (
				<p className="text-sm text-muted-foreground">
					This talk has finished, so it can't be captioned any more.
				</p>
			)}
		</main>
	);
}

function SpeakerControls({ talkId }: { talkId: string }) {
	const speaker = useSpeaker(talkId);
	const on = speaker.status === "live";

	return (
		<>
			<div className="flex flex-col gap-2">
				<div className="flex flex-wrap items-center gap-4">
					<Button
						size="lg"
						variant={on ? "outline" : "default"}
						disabled={speaker.status === "starting"}
						onClick={on ? speaker.stop : speaker.start}
					>
						{on ? <MicOff aria-hidden /> : <Mic aria-hidden />}
						{on ? "Stop captioning" : "Start captioning"}
					</Button>
					<p role="status" className="text-sm text-muted-foreground">
						{speakerStatusText[speaker.status]}
					</p>
				</div>
				{speaker.error && (
					<p role="alert" className="text-sm text-destructive">
						{speaker.error}
					</p>
				)}
			</div>
			<LiveCaptions talkId={talkId} />
		</>
	);
}
