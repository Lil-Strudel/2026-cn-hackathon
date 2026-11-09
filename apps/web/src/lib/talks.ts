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

export type NewTalk = Pick<Talk, "title" | "speaker" | "description">;

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
	/** For transcript hits, the matching caption line and when it was said. */
	lineId?: string;
	at?: string;
};

/** Written by an LLM from the transcript once a talk ends; null until then. */
export type TalkNotes = {
	summary: string | null;
	notes: string[] | null;
};

export type SearchHit = {
	talk: Talk;
	highlights: Highlight[];
};

async function getJSON<T>(path: string): Promise<T> {
	const res = await fetch(path);
	if (!res.ok) throw new Error(`GET ${path}: ${res.status}`);
	return res.json();
}

export async function fetchTalks(): Promise<Talk[]> {
	return getJSON("/api/talks");
}

export async function fetchTalk(id: string): Promise<Talk> {
	return getJSON(`/api/talks/${encodeURIComponent(id)}`);
}

export async function fetchNotes(id: string): Promise<TalkNotes> {
	return getJSON(`/api/talks/${encodeURIComponent(id)}/notes`);
}

export async function searchTalks(query: string): Promise<SearchHit[]> {
	return getJSON(`/api/search?${new URLSearchParams({ q: query })}`);
}

export async function createTalk(talk: NewTalk): Promise<Talk> {
	const res = await fetch("/api/talks", {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(talk),
	});
	if (!res.ok) throw new Error(`POST /api/talks: ${res.status}`);
	return res.json();
}
