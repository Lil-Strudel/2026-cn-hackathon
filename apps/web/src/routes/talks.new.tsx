import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { useId, useState } from "react";
import { Button } from "~/components/ui/button";
import { Card, CardContent } from "~/components/ui/card";
import { Input } from "~/components/ui/input";
import { Label } from "~/components/ui/label";
import { Textarea } from "~/components/ui/textarea";
import { createTalk } from "~/lib/talks";
import { recordTalkCreated } from "~/lib/telemetry";

export const Route = createFileRoute("/talks/new")({
	component: NewTalkPage,
});

function NewTalkPage() {
	const titleId = useId();
	const speakerId = useId();
	const descriptionId = useId();
	const [title, setTitle] = useState("");
	const [speaker, setSpeaker] = useState("");
	const [description, setDescription] = useState("");
	const navigate = Route.useNavigate();
	const queryClient = useQueryClient();

	const creation = useMutation({
		mutationFn: createTalk,
		onSuccess: (talk) => {
			recordTalkCreated(talk.id);
			queryClient.setQueryData(["talk", talk.id], talk);
			void queryClient.invalidateQueries({ queryKey: ["talks"] });
			void navigate({
				to: "/talks/$talkId/speak",
				params: { talkId: talk.id },
			});
		},
	});

	return (
		<main className="mx-auto flex max-w-xl flex-col gap-6 p-4 sm:p-8">
			<Button asChild variant="ghost" size="sm" className="self-start">
				<Link to="/">
					<ArrowLeft aria-hidden />
					All talks
				</Link>
			</Button>

			<div className="flex flex-col gap-2">
				<h1 className="text-2xl font-semibold tracking-tight">New talk</h1>
				<p className="text-sm text-muted-foreground">
					The talk goes live as soon as you create it, and you can start
					captioning it straight away.
				</p>
			</div>

			<Card>
				<CardContent>
					<form
						className="flex flex-col gap-4"
						onSubmit={(e) => {
							e.preventDefault();
							creation.mutate({
								title: title.trim(),
								speaker: speaker.trim(),
								description: description.trim(),
							});
						}}
					>
						<div className="flex flex-col gap-2">
							<Label htmlFor={titleId}>Title</Label>
							<Input
								id={titleId}
								value={title}
								required
								maxLength={120}
								onChange={(e) => setTitle(e.target.value)}
							/>
						</div>
						<div className="flex flex-col gap-2">
							<Label htmlFor={speakerId}>Speaker</Label>
							<Input
								id={speakerId}
								value={speaker}
								required
								maxLength={80}
								autoComplete="name"
								onChange={(e) => setSpeaker(e.target.value)}
							/>
						</div>
						<div className="flex flex-col gap-2">
							<Label htmlFor={descriptionId}>Description</Label>
							<Textarea
								id={descriptionId}
								value={description}
								maxLength={1000}
								onChange={(e) => setDescription(e.target.value)}
							/>
						</div>
						<Button
							type="submit"
							className="self-start"
							disabled={creation.isPending || !title.trim() || !speaker.trim()}
						>
							{creation.isPending ? "Creating…" : "Create talk"}
						</Button>
						{creation.isError && (
							<p role="alert" className="text-sm text-destructive">
								Couldn't create the talk: {creation.error.message}
							</p>
						)}
					</form>
				</CardContent>
			</Card>
		</main>
	);
}
