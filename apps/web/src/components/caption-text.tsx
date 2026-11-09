import { type CaptionLine, splitCorrections } from "~/lib/captions";

export function CaptionText({ line }: { line: CaptionLine }) {
	return splitCorrections(line).map(({ text, heard, offset }) =>
		heard ? (
			<mark
				key={offset}
				title={`Corrected from “${heard}”`}
				className="rounded-sm bg-highlight px-0.5 text-highlight-foreground"
			>
				{text}
				<span className="sr-only"> (corrected from “{heard}”)</span>
			</mark>
		) : (
			<span key={offset}>{text}</span>
		),
	);
}
