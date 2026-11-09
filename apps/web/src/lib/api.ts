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

async function request(
	method: string,
	path: string,
	body?: unknown,
): Promise<Response> {
	const res = await fetch(path, {
		method,
		headers:
			body === undefined ? undefined : { "Content-Type": "application/json" },
		body: body === undefined ? undefined : JSON.stringify(body),
	});
	if (!res.ok) {
		const reason = (await res.text()).trim() || res.statusText;
		throw new Error(`${method} ${path}: ${reason}`);
	}
	return res;
}

export async function correctText(text: string): Promise<Correction> {
	return (await request("POST", "/api/corrections", { text })).json();
}

export type Term = {
	term: string;
	except?: string[];
};

export async function addTerms(terms: Term[]): Promise<void> {
	await request("POST", "/api/vocabulary", { terms });
}

export async function listTerms(): Promise<Term[]> {
	const { terms }: { terms: Term[] } = await (
		await request("GET", "/api/vocabulary")
	).json();
	return terms;
}

export async function removeTerm(term: string): Promise<void> {
	await request("DELETE", `/api/vocabulary/${encodeURIComponent(term)}`);
}
