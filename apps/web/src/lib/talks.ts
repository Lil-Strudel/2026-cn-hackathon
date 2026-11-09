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

const maxSlugLength = 48;

/** A URL-safe id from the title that the api accepts as a talk id, unique among `taken`. */
export function talkSlug(title: string, taken: Set<string>): string {
	const base =
		title
			.normalize("NFKD")
			.replace(/[\u0300-\u036f]/g, "")
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, "-")
			.slice(0, maxSlugLength)
			.replace(/^-+|-+$/g, "") || "talk";
	let slug = base;
	for (let n = 2; taken.has(slug); n++) slug = `${base}-${n}`;
	return slug;
}

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

export async function fetchNotes(id: string): Promise<TalkNotes> {
	if (useFixtures) return (await fixtures()).notesFor(id);
	return getJSON(`/api/talks/${encodeURIComponent(id)}/notes`);
}

export async function searchTalks(query: string): Promise<SearchHit[]> {
	if (useFixtures) return (await fixtures()).search(query);
	return getJSON(`/api/search?${new URLSearchParams({ q: query })}`);
}

export async function createTalk(talk: NewTalk): Promise<Talk> {
	if (useFixtures) return (await fixtures()).addTalk(talk);
	const res = await fetch("/api/talks", {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(talk),
	});
	if (!res.ok) throw new Error(`POST /api/talks: ${res.status}`);
	return res.json();
}
