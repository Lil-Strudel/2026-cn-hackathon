import { type CaptionLine, heardText, splitCorrections } from "~/lib/captions";

export function CaptionText({
	line,
	corrected = true,
}: {
	line: CaptionLine;
	corrected?: boolean;
}) {
	if (!corrected) return heardText(line);

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
