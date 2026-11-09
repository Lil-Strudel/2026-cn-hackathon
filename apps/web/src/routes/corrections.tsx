import { useMutation } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useId, useState } from "react";
import { CorrectedText } from "~/components/corrected-text";
import { Button } from "~/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "~/components/ui/card";
import { Input } from "~/components/ui/input";
import { Label } from "~/components/ui/label";
import { Textarea } from "~/components/ui/textarea";
import { addTerms, correctText } from "~/lib/api";
import { recordCorrection } from "~/lib/telemetry";

export const Route = createFileRoute("/corrections")({
	component: Corrections,
});

const example =
	"we route traffic with silly um and hubble, and scale it on cooper netties with Kedah";

function Corrections() {
	const heardId = useId();
	const termId = useId();
	const [heard, setHeard] = useState(example);
	const [term, setTerm] = useState("");

	const correction = useMutation({
		mutationFn: correctText,
		onSuccess: (result) =>
			recordCorrection(result.segments.filter((s) => s.heard).length),
	});
	const addition = useMutation({
		mutationFn: addTerms,
		onSuccess: () => setTerm(""),
	});

	const corrected =
		correction.data?.segments.filter((s) => s.heard).length ?? 0;

	return (
		<main className="mx-auto flex max-w-xl flex-col gap-6 p-4 sm:p-8">
			<div className="flex flex-col gap-2">
				<h1 className="text-2xl font-semibold tracking-tight">
					Try the caption correction
				</h1>
				<p className="text-sm text-muted-foreground">
					Type what speech recognition might mishear. Terms that sound like a
					word in the vocabulary are replaced and highlighted.
				</p>
			</div>

			<Card>
				<CardContent>
					<form
						className="flex flex-col gap-4"
						onSubmit={(e) => {
							e.preventDefault();
							correction.mutate(heard);
						}}
					>
						<div className="flex flex-col gap-2">
							<Label htmlFor={heardId}>What was heard</Label>
							<Textarea
								id={heardId}
								value={heard}
								maxLength={2000}
								onChange={(e) => setHeard(e.target.value)}
							/>
						</div>
						<Button
							type="submit"
							className="self-start"
							disabled={correction.isPending || !heard.trim()}
						>
							{correction.isPending ? "Correcting…" : "Correct"}
						</Button>
					</form>
				</CardContent>
			</Card>

			<section aria-live="polite" aria-labelledby="result-title">
				<h2 id="result-title" className="sr-only">
					Result
				</h2>
				{correction.isError && (
					<p role="alert" className="text-sm text-destructive">
						{correction.error.message}
					</p>
				)}
				{correction.data && (
					<Card>
						<CardContent className="flex flex-col gap-4">
							<CorrectedText segments={correction.data.segments} />
							<p className="text-sm text-muted-foreground">
								<span className="font-mono tabular-nums">{corrected}</span>{" "}
								{corrected === 1 ? "term" : "terms"} corrected
							</p>
						</CardContent>
					</Card>
				)}
			</section>

			<Card>
				<CardHeader>
					<CardTitle>
						<h2>Add to the vocabulary</h2>
					</CardTitle>
					<CardDescription>
						Names from a talk, like a project or a speaker, so they are spelled
						right in its captions.
					</CardDescription>
				</CardHeader>
				<CardContent>
					<form
						className="flex flex-col gap-2"
						onSubmit={(e) => {
							e.preventDefault();
							addition.mutate([term.trim()]);
						}}
					>
						<Label htmlFor={termId}>Term</Label>
						<div className="flex gap-2">
							<Input
								id={termId}
								value={term}
								maxLength={64}
								onChange={(e) => {
									setTerm(e.target.value);
									addition.reset();
								}}
							/>
							<Button
								type="submit"
								variant="outline"
								disabled={addition.isPending || !term.trim()}
							>
								Add
							</Button>
						</div>
						<p aria-live="polite" className="text-sm">
							{addition.isSuccess && (
								<span className="text-muted-foreground">
									Added. Try it in the box above.
								</span>
							)}
							{addition.isError && (
								<span className="text-destructive">
									{addition.error.message}
								</span>
							)}
						</p>
					</form>
				</CardContent>
			</Card>
		</main>
	);
}
