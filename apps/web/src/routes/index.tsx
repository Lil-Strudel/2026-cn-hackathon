import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { Button } from "~/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "~/components/ui/card";
import { fetchHello } from "~/lib/api";

export const Route = createFileRoute("/")({
	component: Home,
});

function Home() {
	const hello = useQuery({ queryKey: ["hello"], queryFn: fetchHello });

	return (
		<main className="mx-auto flex max-w-xl flex-col gap-6 p-8">
			<Card>
				<CardHeader>
					<CardTitle>Cloud Native Hackathon 2026</CardTitle>
					<CardDescription>
						Boilerplate is live. The product name is decided on the day.
					</CardDescription>
				</CardHeader>
				<CardContent className="flex flex-col gap-4">
					{hello.isPending && (
						<p className="text-muted-foreground">Calling the api…</p>
					)}
					{hello.isError && (
						<p className="text-destructive">{hello.error.message}</p>
					)}
					{hello.data && (
						<dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
							<dt className="text-muted-foreground">api</dt>
							<dd>{hello.data.message}</dd>
							<dt className="text-muted-foreground">elasticsearch</dt>
							<dd>{hello.data.elasticsearch}</dd>
							<dt className="text-muted-foreground">redis</dt>
							<dd>{hello.data.redis}</dd>
						</dl>
					)}
					<Button variant="outline" onClick={() => hello.refetch()}>
						Call again
					</Button>
				</CardContent>
			</Card>
		</main>
	);
}
