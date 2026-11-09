import { Monitor, Moon, Sun } from "lucide-react";
import { ToggleGroup, ToggleGroupItem } from "~/components/ui/toggle-group";
import { useTheme } from "~/hooks/use-theme";
import { isThemePreference } from "~/lib/theme";

const options = [
	{ value: "light", label: "Light", Icon: Sun },
	{ value: "dark", label: "Dark", Icon: Moon },
	{ value: "system", label: "Match system", Icon: Monitor },
] as const;

export function ThemeToggle() {
	const [preference, setPreference] = useTheme();

	return (
		<ToggleGroup
			type="single"
			variant="outline"
			size="sm"
			aria-label="Colour theme"
			value={preference}
			onValueChange={(value) => {
				// Radix reports "" when the active item is clicked again; keep the current choice.
				if (isThemePreference(value)) setPreference(value);
			}}
		>
			{options.map(({ value, label, Icon }) => (
				<ToggleGroupItem
					key={value}
					value={value}
					aria-label={label}
					title={label}
				>
					<Icon />
				</ToggleGroupItem>
			))}
		</ToggleGroup>
	);
}
