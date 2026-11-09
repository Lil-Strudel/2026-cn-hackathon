import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

// Testing Library only cleans up automatically when Vitest globals are on.
afterEach(cleanup);

// jsdom lacks these. Tests run as if the system prefers light.
Object.assign(window, {
	matchMedia: (media: string) => ({
		matches: false,
		media,
		addEventListener: () => {},
		removeEventListener: () => {},
	}),
	scrollTo: () => {},
});
