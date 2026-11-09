export type HighlightPart = { text: string; match: boolean; offset: number };

export function splitHighlight(fragment: string): HighlightPart[] {
	let offset = 0;
	return fragment
		.split(/<mark>(.*?)<\/mark>/g)
		.map((text, index) => {
			const part = { text, match: index % 2 === 1, offset };
			offset += text.length;
			return part;
		})
		.filter((part) => part.text !== "");
}
