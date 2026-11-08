import { init } from "@dash0/sdk-web";

export function initTelemetry() {
	const url = import.meta.env.VITE_DASH0_ENDPOINT;
	const authToken = import.meta.env.VITE_DASH0_AUTH_TOKEN;
	if (!url || !authToken) return;

	init({
		serviceName: "web",
		endpoint: { url, authToken, dataset: import.meta.env.VITE_DASH0_DATASET },
		environment: import.meta.env.MODE,
	});
}
