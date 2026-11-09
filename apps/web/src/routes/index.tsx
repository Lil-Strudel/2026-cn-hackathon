import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Plus, Search } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { HighlightedText } from "~/components/highlighted-text";
import { TalkCard } from "~/components/talk-card";
import { Button } from "~/components/ui/button";
import { Input } from "~/components/ui/input";
import { formatClock } from "~/lib/format";
import type { Highlight, HighlightField, Talk } from "~/lib/talks";
import { fetchTalks, searchTalks } from "~/lib/talks";
import { recordSearch } from "~/lib/telemetry";
import { lineAnchor } from "~/lib/transcript";

export const Route = createFileRoute("/")({
	validateSearch: (search: Record<string, unknown>): { q?: string } => ({
		q: typeof search.q === "string" && search.q !== "" ? search.q : undefined,
	}),
	component: Home,
});

function Home() {
	const { q = "" } = Route.useSearch();
	const talks = useQuery({ queryKey: ["talks"], queryFn: fetchTalks });

	return (
		<main className="mx-auto flex max-w-2xl flex-col gap-10 p-4 sm:p-8">
			<div className="flex flex-wrap items-start justify-between gap-4">
				<div className="flex flex-col gap-2">
					<h1 className="text-2xl font-semibold tracking-tight">Talks</h1>
					<p className="text-sm text-muted-foreground">
						Follow a talk with live captions, or search everything that has been
						said.
					</p>
				</div>
				<Button asChild>
					<Link to="/talks/new">
						<Plus aria-hidden />
						New talk
					</Link>
				</Button>
			</div>

			{talks.isError ? (
				<LoadError error={talks.error} retry={() => talks.refetch()} />
			) : (
				<LiveTalks talks={talks.data} />
			)}

			<section aria-labelledby="search-heading" className="flex flex-col gap-6">
				<SearchBox query={q} />
				{q ? (
					<SearchResults query={q} />
				) : (
					!talks.isError && <PastTalks talks={talks.data} />
				)}
			</section>
		</main>
	);
}

function LiveTalks({ talks }: { talks: Talk[] | undefined }) {
	const live = talks?.filter((talk) => talk.status === "live");

	return (
		<section aria-labelledby="live-heading" className="flex flex-col gap-4">
			<h2 id="live-heading" className="text-lg font-semibold">
				Live now
			</h2>
			{live === undefined && <Loading />}
			{live?.length === 0 && (
				<p className="text-sm text-muted-foreground">
					Nothing is live right now.
				</p>
			)}
			{live?.map((talk) => (
				<TalkCard key={talk.id} talk={talk} />
			))}
		</section>
	);
}

function PastTalks({ talks }: { talks: Talk[] | undefined }) {
	const past = talks
		?.filter((talk) => talk.status === "finished")
		.sort((a, b) => b.startedAt.localeCompare(a.startedAt));

	return (
		<div className="flex flex-col gap-4">
			<h3 className="text-sm font-medium text-muted-foreground">Past talks</h3>
			{past === undefined && <Loading />}
			{past?.length === 0 && (
				<p className="text-sm text-muted-foreground">
					No talks have ended yet.
				</p>
			)}
			{past?.map((talk) => (
				<TalkCard key={talk.id} talk={talk} />
			))}
		</div>
	);
}

const searchDelayMs = 250;

function SearchBox({ query }: { query: string }) {
	const navigate = Route.useNavigate();
	const inputId = useId();
	const [draft, setDraft] = useState(query);

	useEffect(
		() => setDraft((current) => (current.trim() === query ? current : query)),
		[query],
	);

	useEffect(() => {
		const next = draft.trim();
		if (next === query) return;
		const timer = setTimeout(
			() => navigate({ search: { q: next || undefined }, replace: true }),
			searchDelayMs,
		);
		return () => clearTimeout(timer);
	}, [draft, query, navigate]);

	return (
		<search className="flex flex-col gap-2">
			<h2 id="search-heading" className="text-lg font-semibold">
				<label htmlFor={inputId}>Search talks</label>
			</h2>
			<div className="relative">
				<Search
					aria-hidden
					className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground"
				/>
				<Input
					id={inputId}
					type="search"
					value={draft}
					onChange={(event) => setDraft(event.target.value)}
					placeholder="Titles, notes and transcripts"
					className="h-11 bg-card pl-9 md:text-base"
				/>
			</div>
		</search>
	);
}

const fieldLabels: Record<HighlightField, string> = {
	title: "Title",
	description: "Description",
	notes: "Notes",
	summary: "Summary",
	transcript: "Transcript",
};

function SearchResults({ query }: { query: string }) {
	const results = useQuery({
		queryKey: ["search", query],
		queryFn: () => searchTalks(query),
		placeholderData: keepPreviousData,
	});

	useEffect(() => {
		if (results.data && !results.isPlaceholderData)
			recordSearch(query.length, results.data.length);
	}, [results.data, results.isPlaceholderData, query]);

	if (results.isError)
		return <LoadError error={results.error} retry={() => results.refetch()} />;

	return (
		<div className="flex flex-col gap-4">
			<p role="status" className="text-sm text-muted-foreground">
				{results.data === undefined || results.isPlaceholderData
					? "Searching…"
					: results.data.length === 0
						? `No talks mention “${query}”.`
						: `${results.data.length} ${results.data.length === 1 ? "talk matches" : "talks match"} “${query}”.`}
			</p>
			{results.data?.map(({ talk, highlights }) => (
				<TalkCard key={talk.id} talk={talk}>
					<ul className="flex flex-col gap-2 text-sm">
						{highlights
							.filter((highlight) => highlight.field !== "title")
							.map((highlight) => (
								<li key={`${highlight.field}-${highlight.fragment}`}>
									<HighlightSource talkId={talk.id} highlight={highlight} />
									<HighlightedText fragment={highlight.fragment} />
								</li>
							))}
					</ul>
				</TalkCard>
			))}
		</div>
	);
}

function HighlightSource({
	talkId,
	highlight,
}: {
	talkId: string;
	highlight: Highlight;
}) {
	const label = fieldLabels[highlight.field];
	if (!highlight.lineId || !highlight.at)
		return <span className="font-medium">{label}: </span>;

	// `relative` lifts the link above the card's stretched title link.
	return (
		<>
			<Link
				to="/talks/$talkId"
				params={{ talkId }}
				hash={lineAnchor(highlight.lineId)}
				className="relative rounded-sm font-medium text-primary underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
			>
				{label} at{" "}
				<time dateTime={highlight.at} className="font-mono tabular-nums">
					{formatClock(highlight.at)}
				</time>
			</Link>{" "}
		</>
	);
}

function Loading() {
	return <p className="text-sm text-muted-foreground">Loading talks…</p>;
}

function LoadError({ error, retry }: { error: Error; retry: () => void }) {
	return (
		<div role="alert" className="flex flex-col items-start gap-2">
			<p className="text-sm text-destructive">
				Couldn't load talks: {error.message}
			</p>
			<Button variant="outline" size="sm" onClick={retry}>
				Try again
			</Button>
		</div>
	);
}
