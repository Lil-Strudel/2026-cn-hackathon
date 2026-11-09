import { Link } from "@tanstack/react-router";
import type * as React from "react";
import { LiveBadge } from "~/components/live-badge";
import { Card } from "~/components/ui/card";
import { formatDateTime, formatTime } from "~/lib/format";
import type { Talk } from "~/lib/talks";

export function TalkCard({
	talk,
	children,
}: {
	talk: Talk;
	children?: React.ReactNode;
}) {
	const live = talk.status === "live";

	return (
		<Card className="relative gap-3 px-6 py-5 transition-colors hover:bg-accent">
			<div className="flex items-start justify-between gap-4">
				<div className="flex flex-col gap-1">
					<h3 className="text-lg leading-snug font-semibold">
						{/* The link covers the whole card, so anything else clickable inside needs `relative`. */}
						<Link
							to="/talks/$talkId"
							params={{ talkId: talk.id }}
							className="outline-none after:absolute after:inset-0 after:rounded-xl focus-visible:after:ring-3 focus-visible:after:ring-ring focus-visible:after:ring-offset-2 focus-visible:after:ring-offset-background"
						>
							{talk.title}
						</Link>
					</h3>
					<p className="text-sm text-muted-foreground">
						{talk.speaker} ·{" "}
						<time dateTime={talk.startedAt} className="tabular-nums">
							{live
								? `Started ${formatTime(talk.startedAt)}`
								: formatDateTime(talk.startedAt)}
						</time>
					</p>
				</div>
				{live && <LiveBadge />}
			</div>
			{children}
		</Card>
	);
}
