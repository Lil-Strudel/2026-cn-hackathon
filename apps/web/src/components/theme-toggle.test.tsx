import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test } from "vitest";
import { ThemeToggle } from "./theme-toggle";

afterEach(() => localStorage.clear());

test("choosing dark applies and remembers it", () => {
	render(<ThemeToggle />);
	fireEvent.click(screen.getByRole("radio", { name: "Dark" }));
	expect(document.documentElement.classList.contains("dark")).toBe(true);
	expect(localStorage.getItem("theme")).toBe("dark");
});
