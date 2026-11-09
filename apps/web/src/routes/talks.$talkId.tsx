import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { LiveBadge } from "~/components/live-badge";
import { Button } from "~/components/ui/button";
import { formatDateTime } from "~/lib/format";
import { fetchTalk } from "~/lib/talks";

export const Route = createFileRoute("/talks/$talkId")({
	component: TalkPage,
});

function TalkPage() {
	const { talkId } = Route.useParams();
	const talk = useQuery({
		queryKey: ["talk", talkId],
		queryFn: () => fetchTalk(talkId),
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
							{formatDateTime(talk.data.startedAt)}
						</time>
					</div>
					<h1 className="text-2xl font-semibold tracking-tight">
						{talk.data.title}
					</h1>
					<p className="max-w-[65ch] text-sm text-muted-foreground">
						{talk.data.description}
					</p>
				</header>
			)}
		</main>
	);
}
