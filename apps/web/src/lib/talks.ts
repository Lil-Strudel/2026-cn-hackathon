export type TalkStatus = "live" | "finished";

export type Talk = {
	id: string;
	title: string;
	speaker: string;
	description: string;
	status: TalkStatus;
	startedAt: string;
	endedAt: string | null;
};

export type HighlightField =
	| "title"
	| "description"
	| "notes"
	| "summary"
	| "transcript";

export type Highlight = {
	field: HighlightField;
	/** Elasticsearch highlight fragment: plain text with matches wrapped in `<mark>`. */
	fragment: string;
	/** For transcript hits, when the matching line was said. */
	at?: string;
};

export type SearchHit = {
	talk: Talk;
	highlights: Highlight[];
};

// Until the api serves talks, the UI runs on fixtures so it can be built and demoed.
const useFixtures = import.meta.env.VITE_API_FIXTURES !== "false";
const fixtures = () => import("~/lib/talks-fixtures");

async function getJSON<T>(path: string): Promise<T> {
	const res = await fetch(path);
	if (!res.ok) throw new Error(`GET ${path}: ${res.status}`);
	return res.json();
}

export async function fetchTalks(): Promise<Talk[]> {
	if (useFixtures) return (await fixtures()).talks;
	return getJSON("/api/talks");
}

export async function fetchTalk(id: string): Promise<Talk> {
	if (useFixtures) return (await fixtures()).findTalk(id);
	return getJSON(`/api/talks/${encodeURIComponent(id)}`);
}

export async function searchTalks(query: string): Promise<SearchHit[]> {
	if (useFixtures) return (await fixtures()).search(query);
	return getJSON(`/api/search?${new URLSearchParams({ q: query })}`);
}
