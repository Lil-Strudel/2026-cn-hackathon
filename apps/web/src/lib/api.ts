export type Hello = {
	message: string;
	elasticsearch: string;
	redis: string;
};

export async function fetchHello(): Promise<Hello> {
	const res = await fetch("/api/hello");
	if (!res.ok) throw new Error(`GET /api/hello: ${res.status}`);
	return res.json();
}

export type Segment = {
	text: string;
	heard?: string;
};

export type Correction = {
	text: string;
	segments: Segment[];
};

async function post(path: string, body: unknown): Promise<Response> {
	const res = await fetch(path, {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(body),
	});
	if (!res.ok) {
		const reason = (await res.text()).trim() || res.statusText;
		throw new Error(`POST ${path}: ${reason}`);
	}
	return res;
}

export async function correctText(text: string): Promise<Correction> {
	return (await post("/api/corrections", { text })).json();
}

export type Term = {
	term: string;
	context?: string[];
};

export async function addTerms(terms: Term[]): Promise<void> {
	await post("/api/vocabulary", { terms });
}
