import type { QueryClient } from "@tanstack/react-query";
import {
	createRootRouteWithContext,
	Link,
	Outlet,
} from "@tanstack/react-router";
import { ThemeToggle } from "~/components/theme-toggle";
import { Button } from "~/components/ui/button";

const navLinks = [
	{ to: "/", label: "Home" },
	{ to: "/corrections", label: "Try correction" },
] as const;

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()(
	{
		component: () => (
			<div className="min-h-dvh bg-background text-foreground">
				<header className="flex items-center justify-between gap-4 px-4 pt-4 sm:px-8">
					<nav aria-label="Main" className="flex gap-1">
						{navLinks.map(({ to, label }) => (
							<Button key={to} asChild variant="ghost" size="sm">
								<Link
									to={to}
									activeOptions={{ exact: true }}
									activeProps={{
										"aria-current": "page",
										className: "bg-accent",
									}}
								>
									{label}
								</Link>
							</Button>
						))}
					</nav>
					<ThemeToggle />
				</header>
				<Outlet />
			</div>
		),
	},
);
