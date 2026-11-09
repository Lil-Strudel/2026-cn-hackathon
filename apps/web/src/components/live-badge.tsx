import { Badge } from "~/components/ui/badge";

export function LiveBadge() {
	return (
		<Badge variant="live">
			<span
				aria-hidden
				className="size-1.5 rounded-full bg-current motion-safe:animate-pulse"
			/>
			Live
		</Badge>
	);
}
