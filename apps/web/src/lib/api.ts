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
