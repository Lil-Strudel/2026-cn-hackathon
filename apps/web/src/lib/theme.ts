export const themePreferences = ["light", "dark", "system"] as const;
export type ThemePreference = (typeof themePreferences)[number];

export const systemDarkQuery = "(prefers-color-scheme: dark)";
const storageKey = "theme";

export function isThemePreference(value: unknown): value is ThemePreference {
	return themePreferences.includes(value as ThemePreference);
}

export function readPreference(): ThemePreference {
	try {
		const stored = localStorage.getItem(storageKey);
		return isThemePreference(stored) ? stored : "system";
	} catch {
		return "system";
	}
}

export function savePreference(preference: ThemePreference): boolean {
	try {
		if (preference === "system") localStorage.removeItem(storageKey);
		else localStorage.setItem(storageKey, preference);
		return true;
	} catch {
		return false;
	}
}

export function resolveTheme(
	preference: ThemePreference,
	systemPrefersDark: boolean,
): "light" | "dark" {
	if (preference !== "system") return preference;
	return systemPrefersDark ? "dark" : "light";
}

export function applyTheme(preference: ThemePreference) {
	const theme = resolveTheme(preference, matchMedia(systemDarkQuery).matches);
	const root = document.documentElement;
	root.classList.toggle("dark", theme === "dark");
	root.style.colorScheme = theme;
}
