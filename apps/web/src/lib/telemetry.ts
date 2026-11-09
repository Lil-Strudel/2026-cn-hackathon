import { addSignalAttribute, init, sendEvent } from "@dash0/sdk-web";
import { readPreference, type ThemePreference } from "~/lib/theme";

const themeAttribute = "app.theme.preference";
let enabled = false;

export function initTelemetry() {
	const url = import.meta.env.VITE_DASH0_ENDPOINT;
	const authToken = import.meta.env.VITE_DASH0_AUTH_TOKEN;
	if (!url || !authToken) return;

	init({
		serviceName: "web",
		endpoint: { url, authToken, dataset: import.meta.env.VITE_DASH0_DATASET },
		environment: import.meta.env.MODE,
		additionalSignalAttributes: { [themeAttribute]: readPreference() },
	});
	enabled = true;
}

export function recordThemePreference(preference: ThemePreference) {
	if (!enabled) return;
	addSignalAttribute(themeAttribute, preference);
	sendEvent("theme.changed", {
		title: `Theme set to ${preference}`,
		attributes: { [themeAttribute]: preference },
	});
}

export function recordCorrection(corrected: number) {
	if (!enabled) return;
	sendEvent("caption.corrected", {
		title: `Corrected ${corrected} terms`,
		attributes: { "captions.corrections": corrected },
	});
}

// The query text stays out of telemetry: what people search for is their business.
export function recordSearch(queryLength: number, results: number) {
	if (!enabled) return;
	sendEvent("talks.searched", {
		title: `Search returned ${results} talks`,
		attributes: {
			"app.search.query_length": queryLength,
			"app.search.results": results,
		},
	});
}

export function recordCaptionView(view: string) {
	if (!enabled) return;
	sendEvent("captions.view_changed", {
		title: `Caption view set to ${view}`,
		attributes: { "app.captions.view": view },
	});
}

export function recordFirstCaption(waitMs: number) {
	if (!enabled) return;
	sendEvent("captions.first_line", {
		title: "First caption arrived",
		attributes: { "app.captions.wait_ms": Math.round(waitMs) },
	});
}

export function recordCaptionsReconnecting() {
	if (!enabled) return;
	sendEvent("captions.reconnecting", {
		title: "Caption stream lost, reconnecting",
	});
}
