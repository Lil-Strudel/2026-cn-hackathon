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
