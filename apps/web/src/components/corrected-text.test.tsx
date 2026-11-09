import { render, screen } from "@testing-library/react";
import { expect, test } from "vitest";
import { CorrectedText } from "./corrected-text";

test("marks corrected terms and says what was heard", () => {
	render(
		<CorrectedText
			segments={[
				{ text: "we run " },
				{ text: "Cilium", heard: "silly um" },
				{ text: "." },
			]}
		/>,
	);

	const mark = screen.getByTitle("Corrected from “silly um”");
	expect(mark.tagName).toBe("MARK");
	expect(mark.textContent).toBe("Cilium (corrected from silly um)");
	expect(mark.parentElement?.textContent).toBe(
		"we run Cilium (corrected from silly um).",
	);
});
