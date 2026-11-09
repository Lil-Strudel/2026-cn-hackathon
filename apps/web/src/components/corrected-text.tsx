import type * as React from "react";
import type { Segment } from "~/lib/api";
import { cn } from "~/lib/utils";

function CorrectedText({
	segments,
	className,
	...props
}: React.ComponentProps<"p"> & { segments: Segment[] }) {
	let offset = 0;
	return (
		<p
			data-slot="corrected-text"
			className={cn(
				"max-w-[65ch] text-caption font-normal text-card-foreground",
				className,
			)}
			{...props}
		>
			{segments.map((segment) => {
				const key = offset;
				offset += segment.text.length;
				if (!segment.heard) return <span key={key}>{segment.text}</span>;
				return (
					<mark
						key={key}
						title={`Corrected from “${segment.heard}”`}
						className="rounded-sm bg-highlight px-0.5 text-highlight-foreground"
					>
						{segment.text}
						<span className="sr-only"> (corrected from {segment.heard})</span>
					</mark>
				);
			})}
		</p>
	);
}

export { CorrectedText };
