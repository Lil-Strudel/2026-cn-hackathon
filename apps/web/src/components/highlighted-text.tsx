import { Fragment } from "react";
import { splitHighlight } from "~/lib/highlight";

export function HighlightedText({ fragment }: { fragment: string }) {
	return splitHighlight(fragment).map(({ text, match, offset }) =>
		match ? (
			<mark
				key={offset}
				className="rounded-sm bg-highlight px-0.5 text-highlight-foreground"
			>
				{text}
			</mark>
		) : (
			<Fragment key={offset}>{text}</Fragment>
		),
	);
}
