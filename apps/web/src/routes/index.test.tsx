import { screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { renderRoute } from "~/test/render-route";

afterEach(() => vi.restoreAllMocks());

test("shows what the api reports", async () => {
	vi.spyOn(globalThis, "fetch").mockResolvedValue(
		Response.json({
			message: "hello from api",
			elasticsearch: "9.2.0",
			redis: "PONG",
		}),
	);
	renderRoute("/");
	expect(await screen.findByText("hello from api")).toBeDefined();
	expect(screen.getByText("PONG")).toBeDefined();
});

test("shows the error when the api fails", async () => {
	vi.spyOn(globalThis, "fetch").mockResolvedValue(
		new Response(null, { status: 502 }),
	);
	renderRoute("/");
	expect(await screen.findByText("GET /api/hello: 502")).toBeDefined();
});
