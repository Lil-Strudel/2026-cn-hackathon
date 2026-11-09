import { useEffect, useState } from "react";
import { recordThemePreference } from "~/lib/telemetry";
import {
	applyTheme,
	readPreference,
	savePreference,
	systemDarkQuery,
	type ThemePreference,
} from "~/lib/theme";

export function useTheme() {
	const [preference, setPreference] = useState(readPreference);

	useEffect(() => {
		applyTheme(preference);
		if (preference !== "system") return;

		const media = matchMedia(systemDarkQuery);
		const followSystem = () => applyTheme("system");
		media.addEventListener("change", followSystem);
		return () => media.removeEventListener("change", followSystem);
	}, [preference]);

	function changePreference(next: ThemePreference) {
		savePreference(next);
		setPreference(next);
		recordThemePreference(next);
	}

	return [preference, changePreference] as const;
}
